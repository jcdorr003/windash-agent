package ws

import (
	"context"
	"sync"

	"github.com/jcdorr003/windash-agent/internal/metrics"
	"go.uber.org/zap"
)

// BackpressureBuffer manages a buffered channel with backpressure handling
// Drops oldest samples if the buffer is full to prevent blocking
type BackpressureBuffer struct {
	logger     *zap.SugaredLogger
	buffer     chan *metrics.SampleV1
	bufferSize int
	mu         sync.Mutex
	dropped    uint64
}

// NewBackpressureBuffer creates a new backpressure buffer
func NewBackpressureBuffer(logger *zap.SugaredLogger, size int) *BackpressureBuffer {
	return &BackpressureBuffer{
		logger:     logger,
		buffer:     make(chan *metrics.SampleV1, size),
		bufferSize: size,
	}
}

// Push adds a sample to the buffer, dropping the oldest if full
func (b *BackpressureBuffer) Push(sample *metrics.SampleV1) {
	select {
	case b.buffer <- sample:
		// Successfully added to buffer
	default:
		// Buffer is full - drop oldest and add new
		b.mu.Lock()
		b.dropped++
		droppedCount := b.dropped
		b.mu.Unlock()

		// Try to remove oldest
		select {
		case <-b.buffer:
			// Successfully removed oldest
		default:
			// Should not happen, but handle gracefully
		}

		// Add new sample
		select {
		case b.buffer <- sample:
		default:
			b.logger.Warn("⚠️  Failed to add sample even after dropping oldest")
		}

		if droppedCount%10 == 0 {
			b.logger.Warn("⚠️  Backpressure: dropped samples", "totalDropped", droppedCount)
		}
	}
}

// PopBatch retrieves up to maxCount samples from the buffer
func (b *BackpressureBuffer) PopBatch(ctx context.Context, maxCount int) []*metrics.SampleV1 {
	samples := make([]*metrics.SampleV1, 0, maxCount)

	// Get first sample (blocking)
	select {
	case sample := <-b.buffer:
		samples = append(samples, sample)
	case <-ctx.Done():
		return nil
	}

	// Get additional samples (non-blocking, for batching)
	for i := 1; i < maxCount; i++ {
		select {
		case sample := <-b.buffer:
			samples = append(samples, sample)
		default:
			// No more samples available
			return samples
		}
	}

	return samples
}

// Len returns the current buffer length
func (b *BackpressureBuffer) Len() int {
	return len(b.buffer)
}

// DroppedCount returns the total number of dropped samples
func (b *BackpressureBuffer) DroppedCount() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.dropped
}
