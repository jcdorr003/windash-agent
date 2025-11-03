package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jcdorr003/windash-agent/internal/metrics"
	"go.uber.org/zap"
)

const (
	// WebSocket configuration
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 10 * time.Second
	maxMessageSize = 512 * 1024 // 512 KB

	// Reconnect configuration
	initialBackoff = 1 * time.Second
	maxBackoff     = 2 * time.Minute
	backoffFactor  = 2.0
	jitter         = 0.2

	// Buffer configuration
	bufferSize = 100
	batchSize  = 10
)

// Client manages the WebSocket connection to the WinDash backend
type Client struct {
	apiURL string
	token  string
	hostID string
	logger *zap.SugaredLogger

	conn   *websocket.Conn
	buffer *BackpressureBuffer
}

// NewClient creates a new WebSocket client
func NewClient(apiURL, token, hostID string, logger *zap.SugaredLogger) *Client {
	return &Client{
		apiURL: apiURL,
		token:  token,
		hostID: hostID,
		logger: logger,
		buffer: NewBackpressureBuffer(logger, bufferSize),
	}
}

// Run starts the WebSocket client (reconnects automatically on failure)
func (c *Client) Run(ctx context.Context, sampleChan <-chan *metrics.SampleV1) {
	c.logger.Info("🌐 WebSocket client starting")

	backoff := initialBackoff

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("🌐 WebSocket client stopped")
			return
		default:
		}

		// Connect to WebSocket
		if err := c.connect(ctx); err != nil {
			c.logger.Warn("Failed to connect to WebSocket", "error", err, "retryIn", backoff)

			// Exponential backoff with jitter
			jitteredBackoff := addJitter(backoff, jitter)
			time.Sleep(jitteredBackoff)

			backoff = time.Duration(float64(backoff) * backoffFactor)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		c.logger.Info("✅ Connected to WebSocket")
		backoff = initialBackoff // Reset backoff on successful connection

		// Run send and receive loops
		c.runLoop(ctx, sampleChan)

		// Close connection
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}

		c.logger.Warn("🔄 WebSocket disconnected, reconnecting...")
	}
}

// connect establishes a WebSocket connection
func (c *Client) connect(ctx context.Context) error {
	// Build WebSocket URL with hostID
	u, err := url.Parse(c.apiURL)
	if err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}

	q := u.Query()
	q.Set("hostId", c.hostID)
	u.RawQuery = q.Encode()

	// Set up headers
	header := make(map[string][]string)
	header["Authorization"] = []string{fmt.Sprintf("Bearer %s", c.token)}

	// Create dialer with compression
	dialer := websocket.DefaultDialer
	dialer.EnableCompression = true

	// Connect
	conn, resp, err := dialer.DialContext(ctx, u.String(), header)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("WebSocket dial failed (HTTP %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("WebSocket dial failed: %w", err)
	}

	c.conn = conn
	c.conn.SetReadLimit(maxMessageSize)

	return nil
}

// runLoop manages the send and receive loops
func (c *Client) runLoop(ctx context.Context, sampleChan <-chan *metrics.SampleV1) {
	// Context for this connection
	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start reader goroutine (for control messages and pings)
	go c.readLoop(connCtx, cancel)

	// Start writer goroutine
	go c.writeLoop(connCtx, cancel)

	// Buffer samples from the collector
	go c.bufferSamples(connCtx, sampleChan)

	// Wait for context cancellation
	<-connCtx.Done()
}

// readLoop reads control messages from the server
func (c *Client) readLoop(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.logger.Warn("WebSocket read error", "error", err)
			return
		}

		// Parse control message
		var ctrl ControlMessage
		if err := json.Unmarshal(message, &ctrl); err != nil {
			c.logger.Warn("Failed to parse control message", "error", err)
			continue
		}

		c.handleControlMessage(&ctrl)
	}
}

// writeLoop sends metrics and heartbeats to the server
func (c *Client) writeLoop(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Send close message
			c.conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(writeWait),
			)
			return

		case <-ticker.C:
			// Send ping
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Warn("Failed to send ping", "error", err)
				return
			}
			c.logger.Debug("📡 Sent ping")

		default:
			// Try to send batched samples
			samples := c.buffer.PopBatch(ctx, batchSize)
			if len(samples) > 0 {
				if err := c.sendSamples(samples); err != nil {
					c.logger.Warn("Failed to send samples", "error", err)
					return
				}
				c.logger.Debug("📤 Sent samples", "count", len(samples), "buffered", c.buffer.Len())
			} else {
				// No samples available, sleep briefly
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// bufferSamples reads from the collector channel and buffers samples
func (c *Client) bufferSamples(ctx context.Context, sampleChan <-chan *metrics.SampleV1) {
	for {
		select {
		case <-ctx.Done():
			return
		case sample := <-sampleChan:
			c.buffer.Push(sample)
		}
	}
}

// sendSamples sends a batch of samples to the server
func (c *Client) sendSamples(samples []*metrics.SampleV1) error {
	msg := AgentMessage{
		Type:    "metrics",
		Samples: samples,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal samples: %w", err)
	}

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// handleControlMessage processes control messages from the server
func (c *Client) handleControlMessage(msg *ControlMessage) {
	c.logger.Info("📥 Received control message", "type", msg.Type)

	switch msg.Type {
	case "setRate":
		c.logger.Info("🔧 [TODO] Change metrics interval", "intervalMs", msg.IntervalMs)
		// TODO: Implement runtime interval adjustment
	case "pause":
		c.logger.Info("⏸️  [TODO] Pause metrics collection")
		// TODO: Implement pause
	case "resume":
		c.logger.Info("▶️  [TODO] Resume metrics collection")
		// TODO: Implement resume
	default:
		c.logger.Warn("Unknown control message type", "type", msg.Type)
	}
}

// addJitter adds random jitter to a duration
func addJitter(duration time.Duration, jitter float64) time.Duration {
	multiplier := 1.0 + (rand.Float64()*2-1)*jitter
	return time.Duration(float64(duration) * multiplier)
}
