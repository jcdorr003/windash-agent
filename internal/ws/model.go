package ws

import (
	"time"

	"github.com/jcdorr003/windash-agent/internal/metrics"
)

// ControlMessage represents a message from server to agent
type ControlMessage struct {
	Type string `json:"type"` // e.g., "setRate", "pause", "resume"

	// For setRate command
	IntervalMs int `json:"intervalMs,omitempty"`
}

// AgentMessage wraps messages sent from agent to server
type AgentMessage struct {
	Type    string              `json:"type"` // "metrics", "heartbeat", "status"
	Samples []*metrics.SampleV1 `json:"samples,omitempty"`
}

// StatusMessage represents agent status information
type StatusMessage struct {
	Type      string    `json:"type"` // always "status"
	Version   string    `json:"version"`
	Uptime    int64     `json:"uptime"` // seconds
	Timestamp time.Time `json:"timestamp"`
}
