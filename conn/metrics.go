package conn

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds connection metrics.
type Metrics struct {
	// Packet counts
	PacketsRead    atomic.Int64
	PacketsWritten atomic.Int64
	BytesRead      atomic.Int64
	BytesWritten   atomic.Int64

	// Error counts
	ReadErrors    atomic.Int64
	WriteErrors   atomic.Int64
	DecodeErrors  atomic.Int64
	EncodeErrors  atomic.Int64
	DroppedPackets atomic.Int64

	// Latency (ns)
	lastLatency   atomic.Int64
	averageLatency atomic.Int64
	sampleCount   atomic.Int64

	mu          sync.Mutex
	latencyHist [10]int64 	// Histogram buckets in nanoseconds
}

// RecordRead records a read operation.
func (m *Metrics) RecordRead(bytes int) {
	m.PacketsRead.Add(1)
	m.BytesRead.Add(int64(bytes))
}

// RecordWrite records a write operation.
func (m *Metrics) RecordWrite(bytes int) {
	m.PacketsWritten.Add(1)
	m.BytesWritten.Add(int64(bytes))
}

// RecordLatency records a latency measurement.
func (m *Metrics) RecordLatency(d time.Duration) {
	ns := d.Nanoseconds()
	m.lastLatency.Store(ns)

	m.mu.Lock()
	// Moving average
	count := m.sampleCount.Load()
	if count == 0 {
		m.averageLatency.Store(ns)
	} else {
		avg := m.averageLatency.Load()
		newAvg := avg + (ns-avg)/(count+1)
		m.averageLatency.Store(newAvg)
	}
	m.sampleCount.Add(1)

	// Update histogram (logarithmic buckets)
	bucket := 0
	switch {
	case ns < 1_000:
		bucket = 0 // < 1µs
	case ns < 10_000:
		bucket = 1 // 1-10µs
	case ns < 100_000:
		bucket = 2 // 10-100µs
	case ns < 1_000_000:
		bucket = 3 // 100µs-1ms
	case ns < 10_000_000:
		bucket = 4 // 1-10ms
	case ns < 100_000_000:
		bucket = 5 // 10-100ms
	case ns < 500_000_000:
		bucket = 6 // 100-500ms
	default:
		bucket = 7 // >500ms
	}
	m.latencyHist[bucket]++
	m.mu.Unlock()
}

// Latency returns the last measured latency.
func (m *Metrics) Latency() time.Duration {
	return time.Duration(m.lastLatency.Load())
}

// AverageLatency returns the average latency.
func (m *Metrics) AverageLatency() time.Duration {
	return time.Duration(m.averageLatency.Load())
}

// LatencyHistogram returns the latency histogram.
func (m *Metrics) LatencyHistogram() [10]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.latencyHist
}

// Snapshot returns a snapshot of the metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		PacketsRead:     m.PacketsRead.Load(),
		PacketsWritten:  m.PacketsWritten.Load(),
		BytesRead:       m.BytesRead.Load(),
		BytesWritten:    m.BytesWritten.Load(),
		ReadErrors:      m.ReadErrors.Load(),
		WriteErrors:     m.WriteErrors.Load(),
		DroppedPackets:  m.DroppedPackets.Load(),
		Latency:         m.Latency(),
		AverageLatency:  m.AverageLatency(),
	}
}

// Reset resets all metrics.
func (m *Metrics) Reset() {
	m.PacketsRead.Store(0)
	m.PacketsWritten.Store(0)
	m.BytesRead.Store(0)
	m.BytesWritten.Store(0)
	m.ReadErrors.Store(0)
	m.WriteErrors.Store(0)
	m.DecodeErrors.Store(0)
	m.EncodeErrors.Store(0)
	m.DroppedPackets.Store(0)
	m.lastLatency.Store(0)
	m.averageLatency.Store(0)
	m.sampleCount.Store(0)
	m.mu.Lock()
	for i := range m.latencyHist {
		m.latencyHist[i] = 0
	}
	m.mu.Unlock()
}

// MetricsSnapshot is a snapshot of metrics.
type MetricsSnapshot struct {
	PacketsRead    int64
	PacketsWritten int64
	BytesRead      int64
	BytesWritten   int64
	ReadErrors     int64
	WriteErrors    int64
	DroppedPackets int64
	Latency        time.Duration
	AverageLatency time.Duration
}

// ConnMetrics tracks MineConn metrics.
type ConnMetrics struct {
	Inbound  *Metrics
	Outbound *Metrics
}

// NewConnMetrics creates a new ConnMetrics.
func NewConnMetrics() *ConnMetrics {
	return &ConnMetrics{
		Inbound:  &Metrics{},
		Outbound: &Metrics{},
	}
}
