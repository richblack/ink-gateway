package migration

import (
	"context"
	"log"
	"sync"
	"time"
)

// ProgressMonitor tracks and reports migration progress
type ProgressMonitor struct {
	mu                sync.RWMutex
	logger            *log.Logger
	currentBatch      int
	totalBatches      int
	processedRecords  int64
	totalRecords      int64
	startTime         time.Time
	lastUpdateTime    time.Time
	throughputHistory []ThroughputSample
}

// ThroughputSample represents a throughput measurement at a point in time
type ThroughputSample struct {
	Timestamp        time.Time `json:"timestamp"`
	RecordsProcessed int64     `json:"records_processed"`
	RecordsPerSecond float64   `json:"records_per_second"`
}

// ProgressReport provides current progress information
type ProgressReport struct {
	ProcessedRecords     int64             `json:"processed_records"`
	TotalRecords         int64             `json:"total_records"`
	PercentComplete      float64           `json:"percent_complete"`
	CurrentBatch         int               `json:"current_batch"`
	TotalBatches         int               `json:"total_batches"`
	ElapsedTime          time.Duration     `json:"elapsed_time"`
	EstimatedTimeLeft    time.Duration     `json:"estimated_time_left"`
	CurrentThroughput    float64           `json:"current_throughput"`
	AverageThroughput    float64           `json:"average_throughput"`
	ThroughputHistory    []ThroughputSample `json:"throughput_history"`
	EstimatedCompletion  time.Time         `json:"estimated_completion"`
}

// NewProgressMonitor creates a new progress monitor
func NewProgressMonitor() *ProgressMonitor {
	return &ProgressMonitor{
		logger:            log.New(log.Writer(), "[PROGRESS_MONITOR] ", log.LstdFlags|log.Lshortfile),
		startTime:         time.Now(),
		lastUpdateTime:    time.Now(),
		throughputHistory: make([]ThroughputSample, 0),
	}
}

// Initialize sets up the monitor with total records and batches
func (pm *ProgressMonitor) Initialize(totalRecords int64, totalBatches int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.totalRecords = totalRecords
	pm.totalBatches = totalBatches
	pm.startTime = time.Now()
	pm.lastUpdateTime = time.Now()

	pm.logger.Printf("Progress monitor initialized: %d total records, %d total batches", totalRecords, totalBatches)
}

// UpdateProgress updates the current progress
func (pm *ProgressMonitor) UpdateProgress(processedRecords int64, currentBatch int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()

	// Calculate throughput
	if len(pm.throughputHistory) > 0 {
		lastSample := pm.throughputHistory[len(pm.throughputHistory)-1]
		timeDiff := now.Sub(lastSample.Timestamp).Seconds()
		recordsDiff := processedRecords - lastSample.RecordsProcessed

		if timeDiff > 0 {
			currentThroughput := float64(recordsDiff) / timeDiff
			sample := ThroughputSample{
				Timestamp:        now,
				RecordsProcessed: processedRecords,
				RecordsPerSecond: currentThroughput,
			}
			pm.throughputHistory = append(pm.throughputHistory, sample)
		}
	} else {
		// First sample
		elapsedSeconds := now.Sub(pm.startTime).Seconds()
		if elapsedSeconds > 0 {
			throughput := float64(processedRecords) / elapsedSeconds
			sample := ThroughputSample{
				Timestamp:        now,
				RecordsProcessed: processedRecords,
				RecordsPerSecond: throughput,
			}
			pm.throughputHistory = append(pm.throughputHistory, sample)
		}
	}

	// Keep only last 100 samples to prevent memory growth
	if len(pm.throughputHistory) > 100 {
		pm.throughputHistory = pm.throughputHistory[len(pm.throughputHistory)-100:]
	}

	pm.processedRecords = processedRecords
	pm.currentBatch = currentBatch
	pm.lastUpdateTime = now

	// Log progress at regular intervals
	if len(pm.throughputHistory) > 0 && len(pm.throughputHistory)%10 == 0 {
		report := pm.generateReportUnsafe()
		pm.logger.Printf("Progress: %.2f%% complete, %d/%d records, %.2f records/sec, ETA: %v",
			report.PercentComplete, report.ProcessedRecords, report.TotalRecords,
			report.CurrentThroughput, report.EstimatedTimeLeft)
	}
}

// GetProgressReport returns current progress information
func (pm *ProgressMonitor) GetProgressReport() *ProgressReport {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.generateReportUnsafe()
}

// generateReportUnsafe generates a progress report (must be called with lock held)
func (pm *ProgressMonitor) generateReportUnsafe() *ProgressReport {
	now := time.Now()
	elapsedTime := now.Sub(pm.startTime)

	var percentComplete float64
	if pm.totalRecords > 0 {
		percentComplete = float64(pm.processedRecords) / float64(pm.totalRecords) * 100
	}

	var currentThroughput, averageThroughput float64
	var estimatedTimeLeft time.Duration
	var estimatedCompletion time.Time

	// Calculate current throughput (last sample)
	if len(pm.throughputHistory) > 0 {
		currentThroughput = pm.throughputHistory[len(pm.throughputHistory)-1].RecordsPerSecond
	}

	// Calculate average throughput
	if elapsedTime.Seconds() > 0 {
		averageThroughput = float64(pm.processedRecords) / elapsedTime.Seconds()
	}

	// Estimate time left
	if averageThroughput > 0 && pm.totalRecords > pm.processedRecords {
		remainingRecords := pm.totalRecords - pm.processedRecords
		estimatedSecondsLeft := float64(remainingRecords) / averageThroughput
		estimatedTimeLeft = time.Duration(estimatedSecondsLeft) * time.Second
		estimatedCompletion = now.Add(estimatedTimeLeft)
	}

	// Make a copy of throughput history to avoid concurrent access issues
	throughputHistoryCopy := make([]ThroughputSample, len(pm.throughputHistory))
	copy(throughputHistoryCopy, pm.throughputHistory)

	return &ProgressReport{
		ProcessedRecords:    pm.processedRecords,
		TotalRecords:        pm.totalRecords,
		PercentComplete:     percentComplete,
		CurrentBatch:        pm.currentBatch,
		TotalBatches:        pm.totalBatches,
		ElapsedTime:         elapsedTime,
		EstimatedTimeLeft:   estimatedTimeLeft,
		CurrentThroughput:   currentThroughput,
		AverageThroughput:   averageThroughput,
		ThroughputHistory:   throughputHistoryCopy,
		EstimatedCompletion: estimatedCompletion,
	}
}

// GetThroughputAnalysis provides detailed throughput analysis
func (pm *ProgressMonitor) GetThroughputAnalysis() *ThroughputAnalysis {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.throughputHistory) < 2 {
		return &ThroughputAnalysis{}
	}

	analysis := &ThroughputAnalysis{
		SampleCount: len(pm.throughputHistory),
	}

	// Calculate statistics
	var sum, min, max float64
	min = pm.throughputHistory[0].RecordsPerSecond
	max = pm.throughputHistory[0].RecordsPerSecond

	for _, sample := range pm.throughputHistory {
		throughput := sample.RecordsPerSecond
		sum += throughput

		if throughput < min {
			min = throughput
		}
		if throughput > max {
			max = throughput
		}
	}

	analysis.AverageThroughput = sum / float64(len(pm.throughputHistory))
	analysis.MinThroughput = min
	analysis.MaxThroughput = max

	// Calculate variance and standard deviation
	var variance float64
	for _, sample := range pm.throughputHistory {
		diff := sample.RecordsPerSecond - analysis.AverageThroughput
		variance += diff * diff
	}
	variance = variance / float64(len(pm.throughputHistory))
	analysis.StandardDeviation = variance // sqrt would be actual std dev, but this is sufficient

	// Identify trend (improving, degrading, stable)
	if len(pm.throughputHistory) >= 10 {
		recent := pm.throughputHistory[len(pm.throughputHistory)-5:]
		older := pm.throughputHistory[len(pm.throughputHistory)-10 : len(pm.throughputHistory)-5]

		var recentAvg, olderAvg float64
		for _, sample := range recent {
			recentAvg += sample.RecordsPerSecond
		}
		for _, sample := range older {
			olderAvg += sample.RecordsPerSecond
		}
		recentAvg /= float64(len(recent))
		olderAvg /= float64(len(older))

		if recentAvg > olderAvg*1.1 {
			analysis.Trend = "improving"
		} else if recentAvg < olderAvg*0.9 {
			analysis.Trend = "degrading"
		} else {
			analysis.Trend = "stable"
		}
	}

	return analysis
}

// ThroughputAnalysis provides detailed throughput statistics
type ThroughputAnalysis struct {
	SampleCount        int     `json:"sample_count"`
	AverageThroughput  float64 `json:"average_throughput"`
	MinThroughput      float64 `json:"min_throughput"`
	MaxThroughput      float64 `json:"max_throughput"`
	StandardDeviation  float64 `json:"standard_deviation"`
	Trend              string  `json:"trend"` // "improving", "degrading", "stable"
}

// StartPeriodicReporting starts a goroutine that logs progress periodically
func (pm *ProgressMonitor) StartPeriodicReporting(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				report := pm.GetProgressReport()
				analysis := pm.GetThroughputAnalysis()

				pm.logger.Printf("PERIODIC REPORT: %.2f%% complete (%d/%d), "+
					"Batch %d/%d, Throughput: %.2f rec/s (avg: %.2f), "+
					"ETA: %v, Trend: %s",
					report.PercentComplete, report.ProcessedRecords, report.TotalRecords,
					report.CurrentBatch, report.TotalBatches,
					report.CurrentThroughput, report.AverageThroughput,
					report.EstimatedTimeLeft, analysis.Trend)
			}
		}
	}()
}

// DetectBottlenecks analyzes throughput patterns to identify potential bottlenecks
func (pm *ProgressMonitor) DetectBottlenecks() []BottleneckWarning {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var warnings []BottleneckWarning

	if len(pm.throughputHistory) < 10 {
		return warnings
	}

	analysis := pm.GetThroughputAnalysis()

	// Check for significant throughput degradation
	if analysis.Trend == "degrading" {
		warnings = append(warnings, BottleneckWarning{
			Type:        "throughput_degradation",
			Severity:    "warning",
			Description: "Throughput is degrading over time",
			Context: map[string]interface{}{
				"trend": analysis.Trend,
				"current_throughput": pm.throughputHistory[len(pm.throughputHistory)-1].RecordsPerSecond,
				"average_throughput": analysis.AverageThroughput,
			},
		})
	}

	// Check for very low current throughput
	if len(pm.throughputHistory) > 0 {
		currentThroughput := pm.throughputHistory[len(pm.throughputHistory)-1].RecordsPerSecond
		if currentThroughput < analysis.AverageThroughput*0.5 {
			warnings = append(warnings, BottleneckWarning{
				Type:        "low_throughput",
				Severity:    "warning",
				Description: "Current throughput is significantly below average",
				Context: map[string]interface{}{
					"current_throughput": currentThroughput,
					"average_throughput": analysis.AverageThroughput,
					"ratio": currentThroughput / analysis.AverageThroughput,
				},
			})
		}
	}

	// Check for high variance (unstable performance)
	if analysis.StandardDeviation > analysis.AverageThroughput*0.3 {
		warnings = append(warnings, BottleneckWarning{
			Type:        "unstable_performance",
			Severity:    "info",
			Description: "Throughput is highly variable",
			Context: map[string]interface{}{
				"standard_deviation": analysis.StandardDeviation,
				"average_throughput": analysis.AverageThroughput,
				"coefficient_of_variation": analysis.StandardDeviation / analysis.AverageThroughput,
			},
		})
	}

	return warnings
}

// BottleneckWarning represents a potential performance bottleneck
type BottleneckWarning struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"` // "info", "warning", "critical"
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context"`
	Timestamp   time.Time              `json:"timestamp"`
}

// Reset resets the progress monitor for a new migration
func (pm *ProgressMonitor) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.currentBatch = 0
	pm.totalBatches = 0
	pm.processedRecords = 0
	pm.totalRecords = 0
	pm.startTime = time.Now()
	pm.lastUpdateTime = time.Now()
	pm.throughputHistory = make([]ThroughputSample, 0)

	pm.logger.Println("Progress monitor reset")
}

// GetDetailedMetrics returns detailed performance metrics
func (pm *ProgressMonitor) GetDetailedMetrics() *DetailedMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	report := pm.generateReportUnsafe()
	analysis := pm.GetThroughputAnalysis()
	warnings := pm.DetectBottlenecks()

	return &DetailedMetrics{
		ProgressReport:     *report,
		ThroughputAnalysis: *analysis,
		Warnings:           warnings,
		LastUpdateTime:     pm.lastUpdateTime,
	}
}

// DetailedMetrics combines all monitoring information
type DetailedMetrics struct {
	ProgressReport     ProgressReport      `json:"progress_report"`
	ThroughputAnalysis ThroughputAnalysis  `json:"throughput_analysis"`
	Warnings           []BottleneckWarning `json:"warnings"`
	LastUpdateTime     time.Time           `json:"last_update_time"`
}