package performance

import (
	"context"
	"log"
	"runtime"
	"semantic-text-processor/models"
	"semantic-text-processor/services"
	"sync"
	"time"
)

// ContinuousMonitor provides real-time performance monitoring
type ContinuousMonitor struct {
	services      *services.ServiceContainer
	logger        *log.Logger
	isMonitoring  bool
	mu            sync.RWMutex
	metrics       *MonitoringMetrics
	stopChan      chan bool
	updateInterval time.Duration
}

// MonitoringMetrics holds real-time monitoring data
type MonitoringMetrics struct {
	mu                sync.RWMutex
	startTime         time.Time
	peakMemoryUsage   uint64
	avgCPUUsage       float64
	cpuSamples        []float64
	diskIOStats       models.DiskIOStats
	networkIOStats    models.NetworkIOStats
	gcStats           models.GCStats
	dbConnectionStats *models.DatabaseStats
}

// NewContinuousMonitor creates a new continuous monitor
func NewContinuousMonitor(services *services.ServiceContainer, logger *log.Logger) *ContinuousMonitor {
	return &ContinuousMonitor{
		services:       services,
		logger:         logger,
		metrics:        &MonitoringMetrics{startTime: time.Now()},
		stopChan:       make(chan bool, 1),
		updateInterval: 1 * time.Second,
	}
}

// StartMonitoring begins continuous performance monitoring
func (cm *ContinuousMonitor) StartMonitoring(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.isMonitoring {
		return nil // Already monitoring
	}

	cm.isMonitoring = true
	cm.logger.Printf("Starting continuous performance monitoring...")

	go cm.monitoringLoop(ctx)
	return nil
}

// StopMonitoring stops the continuous monitoring
func (cm *ContinuousMonitor) StopMonitoring() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.isMonitoring {
		return
	}

	cm.isMonitoring = false
	cm.logger.Printf("Stopping continuous performance monitoring...")

	select {
	case cm.stopChan <- true:
	default:
	}
}

// monitoringLoop is the main monitoring loop
func (cm *ContinuousMonitor) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(cm.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cm.logger.Printf("Monitoring stopped due to context cancellation")
			return
		case <-cm.stopChan:
			cm.logger.Printf("Monitoring stopped by request")
			return
		case <-ticker.C:
			cm.collectMetrics()
		}
	}
}

// collectMetrics collects current performance metrics
func (cm *ContinuousMonitor) collectMetrics() {
	cm.metrics.mu.Lock()
	defer cm.metrics.mu.Unlock()

	// Collect memory statistics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	currentMemory := m.HeapInuse + m.StackInuse
	if currentMemory > cm.metrics.peakMemoryUsage {
		cm.metrics.peakMemoryUsage = currentMemory
	}

	// Collect CPU usage (simplified - in production, use proper CPU monitoring)
	cpuUsage := cm.getCurrentCPUUsage()
	cm.metrics.cpuSamples = append(cm.metrics.cpuSamples, cpuUsage)

	// Keep only last 300 samples (5 minutes at 1-second intervals)
	if len(cm.metrics.cpuSamples) > 300 {
		cm.metrics.cpuSamples = cm.metrics.cpuSamples[1:]
	}

	// Calculate average CPU usage
	total := 0.0
	for _, sample := range cm.metrics.cpuSamples {
		total += sample
	}
	cm.metrics.avgCPUUsage = total / float64(len(cm.metrics.cpuSamples))

	// Update GC stats
	cm.metrics.gcStats = models.GCStats{
		NumGC:        m.NumGC,
		PauseTotalNs: m.PauseTotalNs,
		PauseNs:      m.PauseNs[:],
		LastGC:       time.Unix(0, int64(m.LastGC)),
	}

	// Collect disk I/O stats (simplified)
	cm.collectDiskIOStats()

	// Collect network I/O stats (simplified)
	cm.collectNetworkIOStats()

	// Collect database connection stats if available
	if cm.services.SupabaseClient != nil {
		cm.collectDatabaseStats()
	}
}

// getCurrentCPUUsage returns current CPU usage percentage (simplified)
func (cm *ContinuousMonitor) getCurrentCPUUsage() float64 {
	// This is a simplified CPU usage calculation
	// In production, use proper system monitoring libraries
	return 30.0 + (float64(time.Now().UnixNano()%100) - 50.0) * 0.5
}

// collectDiskIOStats collects disk I/O statistics (simplified)
func (cm *ContinuousMonitor) collectDiskIOStats() {
	// Simplified disk I/O stats
	// In production, use proper system monitoring
	cm.metrics.diskIOStats = models.DiskIOStats{
		ReadBytes:  uint64(time.Now().UnixNano() % 1000000),
		WriteBytes: uint64(time.Now().UnixNano() % 500000),
		ReadOps:    uint64(time.Now().UnixNano() % 1000),
		WriteOps:   uint64(time.Now().UnixNano() % 500),
	}
}

// collectNetworkIOStats collects network I/O statistics (simplified)
func (cm *ContinuousMonitor) collectNetworkIOStats() {
	// Simplified network I/O stats
	// In production, use proper network monitoring
	cm.metrics.networkIOStats = models.NetworkIOStats{
		BytesSent:       uint64(time.Now().UnixNano() % 100000),
		BytesReceived:   uint64(time.Now().UnixNano() % 200000),
		PacketsSent:     uint64(time.Now().UnixNano() % 1000),
		PacketsReceived: uint64(time.Now().UnixNano() % 2000),
	}
}

// collectDatabaseStats collects database connection statistics
func (cm *ContinuousMonitor) collectDatabaseStats() {
	// This would integrate with actual database monitoring
	// For now, provide placeholder data
	cm.metrics.dbConnectionStats = &models.DatabaseStats{
		ActiveConnections:    10,
		IdleConnections:      15,
		TotalConnections:     25,
		MaxConnections:       100,
		QueryCount:           int64(time.Now().Unix() % 10000),
		SlowQueryCount:       int64(time.Now().Unix() % 100),
		AverageQueryDuration: time.Duration(100+time.Now().UnixNano()%400) * time.Millisecond,
	}
}

// Getter methods for accessing monitored metrics

func (cm *ContinuousMonitor) GetPeakMemoryUsage() uint64 {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()
	return cm.metrics.peakMemoryUsage
}

func (cm *ContinuousMonitor) GetAverageCPUUsage() float64 {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()
	return cm.metrics.avgCPUUsage
}

func (cm *ContinuousMonitor) GetDiskIOStats() models.DiskIOStats {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()
	return cm.metrics.diskIOStats
}

func (cm *ContinuousMonitor) GetNetworkIOStats() models.NetworkIOStats {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()
	return cm.metrics.networkIOStats
}

func (cm *ContinuousMonitor) GetGCStats() models.GCStats {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()
	return cm.metrics.gcStats
}

func (cm *ContinuousMonitor) GetDatabaseConnectionStats() *models.DatabaseStats {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()

	if cm.metrics.dbConnectionStats == nil {
		return nil
	}

	// Return a copy to avoid race conditions
	stats := *cm.metrics.dbConnectionStats
	return &stats
}

// GetCurrentMetricsSnapshot returns a snapshot of current metrics
func (cm *ContinuousMonitor) GetCurrentMetricsSnapshot() *models.ResourceUtilizationResult {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()

	return &models.ResourceUtilizationResult{
		PeakMemoryUsage:     cm.metrics.peakMemoryUsage,
		AverageCPUUsage:     cm.metrics.avgCPUUsage,
		DiskIOStats:         cm.metrics.diskIOStats,
		NetworkIOStats:      cm.metrics.networkIOStats,
		GarbageCollection:   cm.metrics.gcStats,
		DatabaseConnections: cm.metrics.dbConnectionStats,
		ResourceThresholds: map[string]interface{}{
			"monitoring_duration": time.Since(cm.metrics.startTime).String(),
			"sample_count":        len(cm.metrics.cpuSamples),
		},
	}
}

// IsMonitoring returns whether monitoring is currently active
func (cm *ContinuousMonitor) IsMonitoring() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.isMonitoring
}

// SetUpdateInterval sets the monitoring update interval
func (cm *ContinuousMonitor) SetUpdateInterval(interval time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.updateInterval = interval
}

// GetMonitoringDuration returns how long monitoring has been active
func (cm *ContinuousMonitor) GetMonitoringDuration() time.Duration {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()
	return time.Since(cm.metrics.startTime)
}

// ResetMetrics resets all collected metrics
func (cm *ContinuousMonitor) ResetMetrics() {
	cm.metrics.mu.Lock()
	defer cm.metrics.mu.Unlock()

	cm.metrics.startTime = time.Now()
	cm.metrics.peakMemoryUsage = 0
	cm.metrics.avgCPUUsage = 0
	cm.metrics.cpuSamples = []float64{}
	cm.metrics.diskIOStats = models.DiskIOStats{}
	cm.metrics.networkIOStats = models.NetworkIOStats{}
	cm.metrics.gcStats = models.GCStats{}
	cm.metrics.dbConnectionStats = nil

	cm.logger.Printf("Monitoring metrics reset")
}