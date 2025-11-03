package services

import "time"

// NoOpMonitor is a no-operation implementation of QueryPerformanceMonitor
type NoOpMonitor struct{}

// NewNoOpMonitor creates a new no-op monitor
func NewNoOpMonitor() QueryPerformanceMonitor {
	return &NoOpMonitor{}
}

// RecordQuery does nothing
func (m *NoOpMonitor) RecordQuery(queryType string, duration time.Duration, rowCount int) {
	// No-op
}

// RecordSlowQuery does nothing
func (m *NoOpMonitor) RecordSlowQuery(query string, duration time.Duration, params map[string]interface{}) {
	// No-op
}

// GetQueryStats returns empty stats
func (m *NoOpMonitor) GetQueryStats() QueryStatistics {
	return QueryStatistics{
		QueryTypes: make(map[string]QueryTypeStats),
	}
}

// GetSlowQueries returns empty slow queries
func (m *NoOpMonitor) GetSlowQueries(limit int) []SlowQueryRecord {
	return []SlowQueryRecord{}
}
