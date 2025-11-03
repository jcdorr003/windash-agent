package metrics

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"go.uber.org/zap"
)

// SampleV1 represents a versioned metrics sample
type SampleV1 struct {
	V      int       `json:"v"`  // Schema version (always 1)
	TS     time.Time `json:"ts"` // Timestamp
	HostID string    `json:"hostId"`

	CPU struct {
		Total   float64   `json:"total"`             // Total CPU usage %
		PerCore []float64 `json:"perCore,omitempty"` // Per-core usage %
	} `json:"cpu"`

	Mem struct {
		Used  uint64 `json:"used"`  // Used memory in bytes
		Total uint64 `json:"total"` // Total memory in bytes
	} `json:"mem"`

	Disks []struct {
		Name  string `json:"name"`  // Mount point or drive letter
		Used  uint64 `json:"used"`  // Used space in bytes
		Total uint64 `json:"total"` // Total space in bytes
	} `json:"disk"`

	Net struct {
		TxBps uint64 `json:"txBps"` // Transmit bytes per second
		RxBps uint64 `json:"rxBps"` // Receive bytes per second
	} `json:"net"`

	UptimeSec uint64 `json:"uptimeSec"` // System uptime in seconds
	ProcCount uint64 `json:"procCount"` // Number of running processes
}

// Collector periodically collects system metrics
type Collector struct {
	logger   *zap.SugaredLogger
	hostID   string
	interval time.Duration

	// For network rate calculations
	lastNetStats net.IOCountersStat
	lastNetTime  time.Time
}

// NewCollector creates a new metrics collector
func NewCollector(logger *zap.SugaredLogger, hostID string, interval time.Duration) *Collector {
	return &Collector{
		logger:   logger,
		hostID:   hostID,
		interval: interval,
	}
}

// Start begins collecting metrics and sending them to the channel
func (c *Collector) Start(ctx context.Context, sampleChan chan<- *SampleV1) {
	c.logger.Info("📊 Metrics collector started", "interval", c.interval)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Collect initial sample immediately
	if sample := c.collect(); sample != nil {
		select {
		case sampleChan <- sample:
		case <-ctx.Done():
			return
		}
	}

	for {
		select {
		case <-ticker.C:
			if sample := c.collect(); sample != nil {
				select {
				case sampleChan <- sample:
				case <-ctx.Done():
					return
				default:
					c.logger.Warn("⚠️  Sample channel full, dropping oldest sample")
				}
			}
		case <-ctx.Done():
			c.logger.Info("📊 Metrics collector stopped")
			return
		}
	}
}

// collect gathers all system metrics
func (c *Collector) collect() *SampleV1 {
	sample := &SampleV1{
		V:      1,
		TS:     time.Now(),
		HostID: c.hostID,
	}

	// CPU metrics
	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		sample.CPU.Total = cpuPercent[0]
	}

	if cpuPerCore, err := cpu.Percent(0, true); err == nil {
		sample.CPU.PerCore = cpuPerCore
	}

	// Memory metrics
	if memInfo, err := mem.VirtualMemory(); err == nil {
		sample.Mem.Used = memInfo.Used
		sample.Mem.Total = memInfo.Total
	}

	// Disk metrics
	if partitions, err := disk.Partitions(false); err == nil {
		for _, partition := range partitions {
			if usage, err := disk.Usage(partition.Mountpoint); err == nil {
				sample.Disks = append(sample.Disks, struct {
					Name  string `json:"name"`
					Used  uint64 `json:"used"`
					Total uint64 `json:"total"`
				}{
					Name:  partition.Mountpoint,
					Used:  usage.Used,
					Total: usage.Total,
				})
			}
		}
	}

	// Network metrics (calculate rates)
	if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
		now := time.Now()
		if !c.lastNetTime.IsZero() {
			elapsed := now.Sub(c.lastNetTime).Seconds()
			if elapsed > 0 {
				sample.Net.TxBps = uint64(float64(netStats[0].BytesSent-c.lastNetStats.BytesSent) / elapsed)
				sample.Net.RxBps = uint64(float64(netStats[0].BytesRecv-c.lastNetStats.BytesRecv) / elapsed)
			}
		}
		c.lastNetStats = netStats[0]
		c.lastNetTime = now
	}

	// Uptime
	if uptime, err := host.Uptime(); err == nil {
		sample.UptimeSec = uptime
	}

	// Process count
	if procs, err := process.Pids(); err == nil {
		sample.ProcCount = uint64(len(procs))
	}

	c.logger.Debug("📈 Collected metrics",
		"cpu", sample.CPU.Total,
		"memUsed", sample.Mem.Used,
		"diskCount", len(sample.Disks),
	)

	return sample
}
