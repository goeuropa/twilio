package health

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// MetricsCollector collects and manages health check metrics
type MetricsCollector struct {
	mu                            sync.RWMutex
	healthChecksTotal             int64
	healthChecksDurationTotal     time.Duration
	healthChecksFailuresTotal     int64
	systemStartTime               time.Time
	lastMetricsUpdate             time.Time
	DependencyRequestsTotal       int64
	DependencyErrorsTotal         int64
	DependencyResponseTimeSeconds float64
	SessionStoreSessionsTotal     int64
	SessionStoreCacheHitsTotal    int64
	SessionStoreCacheMissesTotal  int64
	CircuitBreakerState           int
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		systemStartTime: time.Now(),
	}
}

// UpdateMetrics updates metrics based on health check results
func (mc *MetricsCollector) UpdateMetrics(results map[string]CheckResult, totalDuration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.healthChecksTotal++
	mc.healthChecksDurationTotal += totalDuration
	mc.lastMetricsUpdate = time.Now()

	// Count failures
	for _, result := range results {
		if result.Status != StatusHealthy {
			mc.healthChecksFailuresTotal++
		}
	}
}

// GetMetrics returns current metrics information
func (mc *MetricsCollector) GetMetrics() MetricsInfo {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	avgDuration := 0.0
	if mc.healthChecksTotal > 0 {
		avgDuration = float64(mc.healthChecksDurationTotal.Nanoseconds()) / float64(mc.healthChecksTotal) / 1e9
	}

	usagePercent := 0.0
	if memStats.HeapSys > 0 {
		usagePercent = float64(memStats.HeapInuse) / float64(memStats.HeapSys) * 100
	}

	return MetricsInfo{
		HealthChecksTotal:             mc.healthChecksTotal,
		HealthChecksDurationSeconds:   avgDuration,
		HealthChecksFailuresTotal:     mc.healthChecksFailuresTotal,
		DependencyRequestsTotal:       mc.DependencyRequestsTotal,
		DependencyErrorsTotal:         mc.DependencyErrorsTotal,
		DependencyResponseTimeSeconds: mc.DependencyResponseTimeSeconds,
		SessionStoreSessionsTotal:     mc.SessionStoreSessionsTotal,
		SessionStoreCacheHitsTotal:    mc.SessionStoreCacheHitsTotal,
		SessionStoreCacheMissesTotal:  mc.SessionStoreCacheMissesTotal,
		CircuitBreakerState:           mc.CircuitBreakerState,
		SystemGoroutinesTotal:         runtime.NumGoroutine(),
		SystemMemoryAllocBytes:        memStats.Alloc,
		SystemMemoryUsagePercent:      usagePercent,
		SystemUptimeSeconds:           time.Since(mc.systemStartTime).Seconds(),
	}
}

// GetPrometheusMetrics returns metrics in Prometheus format
func (mc *MetricsCollector) GetPrometheusMetrics() string {
	metrics := mc.GetMetrics()

	return formatPrometheusMetrics(metrics)
}

// formatPrometheusMetrics formats metrics for Prometheus
func formatPrometheusMetrics(metrics MetricsInfo) string {
	result := ""

	// Health check metrics
	result += "# HELP health_checks_total Total number of health checks performed\n"
	result += "# TYPE health_checks_total counter\n"
	result += formatMetricLine("health_checks_total", "", float64(metrics.HealthChecksTotal))

	result += "# HELP health_checks_duration_seconds Average duration of health checks in seconds\n"
	result += "# TYPE health_checks_duration_seconds gauge\n"
	result += formatMetricLine("health_checks_duration_seconds", "", metrics.HealthChecksDurationSeconds)

	result += "# HELP health_checks_failures_total Total number of failed health checks\n"
	result += "# TYPE health_checks_failures_total counter\n"
	result += formatMetricLine("health_checks_failures_total", "", float64(metrics.HealthChecksFailuresTotal))

	// Dependency metrics
	if metrics.DependencyRequestsTotal > 0 {
		result += "# HELP dependency_requests_total Total number of dependency requests\n"
		result += "# TYPE dependency_requests_total counter\n"
		result += formatMetricLine("dependency_requests_total", "", float64(metrics.DependencyRequestsTotal))
	}

	if metrics.DependencyErrorsTotal > 0 {
		result += "# HELP dependency_errors_total Total number of dependency errors\n"
		result += "# TYPE dependency_errors_total counter\n"
		result += formatMetricLine("dependency_errors_total", "", float64(metrics.DependencyErrorsTotal))
	}

	if metrics.DependencyResponseTimeSeconds > 0 {
		result += "# HELP dependency_response_time_seconds Average dependency response time in seconds\n"
		result += "# TYPE dependency_response_time_seconds gauge\n"
		result += formatMetricLine("dependency_response_time_seconds", "", metrics.DependencyResponseTimeSeconds)
	}

	// Session store metrics
	if metrics.SessionStoreSessionsTotal > 0 {
		result += "# HELP session_store_sessions_total Total number of active sessions\n"
		result += "# TYPE session_store_sessions_total gauge\n"
		result += formatMetricLine("session_store_sessions_total", "", float64(metrics.SessionStoreSessionsTotal))
	}

	if metrics.SessionStoreCacheHitsTotal > 0 {
		result += "# HELP session_store_cache_hits_total Total number of session store cache hits\n"
		result += "# TYPE session_store_cache_hits_total counter\n"
		result += formatMetricLine("session_store_cache_hits_total", "", float64(metrics.SessionStoreCacheHitsTotal))
	}

	if metrics.SessionStoreCacheMissesTotal > 0 {
		result += "# HELP session_store_cache_misses_total Total number of session store cache misses\n"
		result += "# TYPE session_store_cache_misses_total counter\n"
		result += formatMetricLine("session_store_cache_misses_total", "", float64(metrics.SessionStoreCacheMissesTotal))
	}

	// Circuit breaker metrics
	result += "# HELP circuit_breaker_state Current state of circuit breaker (0=closed, 1=open, 2=half-open)\n"
	result += "# TYPE circuit_breaker_state gauge\n"
	result += formatMetricLine("circuit_breaker_state", "", float64(metrics.CircuitBreakerState))

	// System metrics
	result += "# HELP system_goroutines_total Current number of goroutines\n"
	result += "# TYPE system_goroutines_total gauge\n"
	result += formatMetricLine("system_goroutines_total", "", float64(metrics.SystemGoroutinesTotal))

	result += "# HELP system_memory_alloc_bytes Current bytes allocated\n"
	result += "# TYPE system_memory_alloc_bytes gauge\n"
	result += formatMetricLine("system_memory_alloc_bytes", "", float64(metrics.SystemMemoryAllocBytes))

	result += "# HELP system_memory_usage_percent Current memory usage percentage\n"
	result += "# TYPE system_memory_usage_percent gauge\n"
	result += formatMetricLine("system_memory_usage_percent", "", metrics.SystemMemoryUsagePercent)

	result += "# HELP system_uptime_seconds System uptime in seconds\n"
	result += "# TYPE system_uptime_seconds gauge\n"
	result += formatMetricLine("system_uptime_seconds", "", metrics.SystemUptimeSeconds)

	return result
}

// formatMetricLine formats a single metric line for Prometheus
func formatMetricLine(name, labels string, value float64) string {
	if labels != "" {
		return fmt.Sprintf("%s{%s} %.6f\n", name, labels, value)
	}
	return fmt.Sprintf("%s %.6f\n", name, value)
}

// IncrementDependencyRequests increments dependency request counter
func (mc *MetricsCollector) IncrementDependencyRequests() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.DependencyRequestsTotal++
}

// IncrementDependencyErrors increments dependency error counter
func (mc *MetricsCollector) IncrementDependencyErrors() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.DependencyErrorsTotal++
}

// UpdateDependencyResponseTime updates dependency response time
func (mc *MetricsCollector) UpdateDependencyResponseTime(duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.DependencyResponseTimeSeconds = duration.Seconds()
}

// UpdateSessionStoreMetrics updates session store metrics
func (mc *MetricsCollector) UpdateSessionStoreMetrics(sessions, hits, misses int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.SessionStoreSessionsTotal = sessions
	mc.SessionStoreCacheHitsTotal = hits
	mc.SessionStoreCacheMissesTotal = misses
}

// UpdateCircuitBreakerState updates circuit breaker state
func (mc *MetricsCollector) UpdateCircuitBreakerState(state int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.CircuitBreakerState = state
}
