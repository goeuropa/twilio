package health

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"oba-twilio/client"
	"oba-twilio/handlers/common"
	"oba-twilio/localization"
)

// SystemHealthChecker checks basic system health
type SystemHealthChecker struct{}

func (c *SystemHealthChecker) Name() string {
	return "system"
}

func (c *SystemHealthChecker) Check() CheckResult {
	start := time.Now()

	// Check basic system resources
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	goroutines := runtime.NumGoroutine()

	// Define thresholds
	const (
		maxGoroutines    = 10000
		maxMemoryMB      = 1024 // 1GB
		maxMemoryPercent = 90.0
	)

	status := StatusHealthy
	message := "System is healthy"
	metadata := map[string]string{
		"goroutines":   fmt.Sprintf("%d", goroutines),
		"memory_alloc": fmt.Sprintf("%.2f MB", float64(memStats.Alloc)/1024/1024),
		"memory_sys":   fmt.Sprintf("%.2f MB", float64(memStats.Sys)/1024/1024),
		"gc_cycles":    fmt.Sprintf("%d", memStats.NumGC),
	}

	// Check goroutine count
	if goroutines > maxGoroutines {
		status = StatusDegraded
		message = fmt.Sprintf("High goroutine count: %d", goroutines)
	}

	// Check memory usage
	memoryMB := float64(memStats.Alloc) / 1024 / 1024
	if memoryMB > maxMemoryMB {
		status = StatusDegraded
		message = fmt.Sprintf("High memory usage: %.2f MB", memoryMB)
	}

	// Check memory percentage
	if memStats.HeapSys > 0 {
		usagePercent := float64(memStats.HeapInuse) / float64(memStats.HeapSys) * 100
		metadata["memory_usage_percent"] = fmt.Sprintf("%.2f", usagePercent)

		if usagePercent > maxMemoryPercent {
			status = StatusDegraded
			message = fmt.Sprintf("High memory usage: %.2f%%", usagePercent)
		}
	}

	return CheckResult{
		Name:      c.Name(),
		Status:    status,
		Message:   message,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
}

// OneBusAwayHealthChecker checks OneBusAway API connectivity
type OneBusAwayHealthChecker struct {
	client *client.OneBusAwayClient
}

func NewOneBusAwayHealthChecker(obaClient *client.OneBusAwayClient) *OneBusAwayHealthChecker {
	return &OneBusAwayHealthChecker{client: obaClient}
}

func (c *OneBusAwayHealthChecker) Name() string {
	return "onebusaway_api"
}

func (c *OneBusAwayHealthChecker) Check() CheckResult {
	start := time.Now()

	if c.client == nil {
		return CheckResult{
			Name:      c.Name(),
			Status:    StatusUnhealthy,
			Error:     "OneBusAway client is not initialized",
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}

	// Get client metrics for health assessment
	metrics := c.client.GetMetrics()

	status := StatusHealthy
	message := "OneBusAway API is healthy"
	errorMsg := ""

	metadata := map[string]string{
		"cache_hits":        fmt.Sprintf("%d", metrics.CacheHits),
		"cache_misses":      fmt.Sprintf("%d", metrics.CacheMisses),
		"api_calls":         fmt.Sprintf("%d", metrics.APICallCount),
		"api_errors":        fmt.Sprintf("%d", metrics.APIErrorCount),
		"circuit_breaker":   fmt.Sprintf("%d", metrics.CircuitBreakerOpen),
		"validation_errors": fmt.Sprintf("%d", metrics.ValidationErrors),
	}

	// Calculate error rate
	errorRate := 0.0
	if metrics.APICallCount > 0 {
		errorRate = float64(metrics.APIErrorCount) / float64(metrics.APICallCount) * 100
		metadata["error_rate_percent"] = fmt.Sprintf("%.2f", errorRate)
	}

	// Calculate cache hit rate
	if metrics.CacheHits+metrics.CacheMisses > 0 {
		hitRate := float64(metrics.CacheHits) / float64(metrics.CacheHits+metrics.CacheMisses) * 100
		metadata["cache_hit_rate_percent"] = fmt.Sprintf("%.2f", hitRate)
	}

	// Average response time
	if metrics.APICallCount > 0 {
		avgResponseTime := metrics.TotalResponseTime / time.Duration(metrics.APICallCount)
		metadata["avg_response_time"] = avgResponseTime.String()

		// Check if response time is too high
		if avgResponseTime > 5*time.Second {
			status = StatusDegraded
			message = fmt.Sprintf("High API response time: %s", avgResponseTime)
		}
	}

	// Check error rate threshold
	if errorRate > 10.0 {
		status = StatusDegraded
		message = fmt.Sprintf("High API error rate: %.2f%%", errorRate)
	} else if errorRate > 25.0 {
		status = StatusUnhealthy
		errorMsg = fmt.Sprintf("Critical API error rate: %.2f%%", errorRate)
	}

	// Check circuit breaker status
	if metrics.CircuitBreakerOpen > 0 {
		status = StatusDegraded
		message = fmt.Sprintf("Circuit breaker has opened %d times", metrics.CircuitBreakerOpen)
	}

	// Test basic connectivity with coverage check
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Try to get coverage area as a health check
	coverageArea := c.client.GetCoverageArea()
	if coverageArea == nil {
		// Try to initialize coverage as a connectivity test
		if err := c.client.InitializeCoverage(); err != nil {
			status = StatusUnhealthy
			errorMsg = fmt.Sprintf("Cannot connect to OneBusAway API: %v", err)
		}
	} else {
		metadata["coverage_center_lat"] = fmt.Sprintf("%.4f", coverageArea.CenterLat)
		metadata["coverage_center_lon"] = fmt.Sprintf("%.4f", coverageArea.CenterLon)
		metadata["coverage_radius"] = fmt.Sprintf("%.0f", coverageArea.Radius)
	}

	_ = ctx // Avoid unused variable warning

	result := CheckResult{
		Name:      c.Name(),
		Status:    status,
		Message:   message,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	if errorMsg != "" {
		result.Error = errorMsg
	}

	return result
}

// SessionStoreHealthChecker checks session store health
type SessionStoreHealthChecker struct {
	store *common.ImprovedSessionStore
}

func NewSessionStoreHealthChecker(sessionStore *common.ImprovedSessionStore) *SessionStoreHealthChecker {
	return &SessionStoreHealthChecker{store: sessionStore}
}

func (c *SessionStoreHealthChecker) Name() string {
	return "session_store"
}

func (c *SessionStoreHealthChecker) Check() CheckResult {
	start := time.Now()

	if c.store == nil {
		return CheckResult{
			Name:      c.Name(),
			Status:    StatusUnhealthy,
			Error:     "Session store is not initialized",
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}

	// Get session store metrics
	metrics := c.store.GetMetrics()

	status := StatusHealthy
	message := "Session store is healthy"

	metadata := map[string]string{
		"total_sessions":     fmt.Sprintf("%d", metrics.TotalSessions),
		"cache_hits":         fmt.Sprintf("%d", metrics.CacheHits),
		"cache_misses":       fmt.Sprintf("%d", metrics.CacheMisses),
		"evictions":          fmt.Sprintf("%d", metrics.Evictions),
		"expired_sessions":   fmt.Sprintf("%d", metrics.ExpiredSessions),
		"created_sessions":   fmt.Sprintf("%d", metrics.CreatedSessions),
		"memory_usage_bytes": fmt.Sprintf("%d", metrics.MemoryUsage),
	}

	// Calculate cache hit rate
	if metrics.CacheHits+metrics.CacheMisses > 0 {
		hitRate := float64(metrics.CacheHits) / float64(metrics.CacheHits+metrics.CacheMisses) * 100
		metadata["cache_hit_rate_percent"] = fmt.Sprintf("%.2f", hitRate)

		// Check if cache hit rate is too low
		if hitRate < 50.0 {
			status = StatusDegraded
			message = fmt.Sprintf("Low session store cache hit rate: %.2f%%", hitRate)
		}
	}

	// Check memory usage (convert to MB)
	memoryMB := float64(metrics.MemoryUsage) / 1024 / 1024
	metadata["memory_usage_mb"] = fmt.Sprintf("%.2f", memoryMB)

	// Check for high memory usage
	if memoryMB > 100 { // 100MB threshold
		status = StatusDegraded
		message = fmt.Sprintf("High session store memory usage: %.2f MB", memoryMB)
	}

	// Check session count
	if metrics.TotalSessions > 5000 {
		status = StatusDegraded
		message = fmt.Sprintf("High session count: %d", metrics.TotalSessions)
	}

	// Add last cleanup time if available
	if metrics.LastCleanupTime > 0 {
		lastCleanup := time.Unix(metrics.LastCleanupTime, 0)
		metadata["last_cleanup"] = lastCleanup.Format(time.RFC3339)

		// Check if cleanup is stale (older than 30 minutes)
		if time.Since(lastCleanup) > 30*time.Minute {
			status = StatusDegraded
			message = "Session store cleanup is overdue"
		}
	}

	return CheckResult{
		Name:      c.Name(),
		Status:    status,
		Message:   message,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
}

// LocalizationHealthChecker checks localization system health
type LocalizationHealthChecker struct {
	manager *localization.LocalizationManager
}

func NewLocalizationHealthChecker(locManager *localization.LocalizationManager) *LocalizationHealthChecker {
	return &LocalizationHealthChecker{manager: locManager}
}

func (c *LocalizationHealthChecker) Name() string {
	return "localization"
}

func (c *LocalizationHealthChecker) Check() CheckResult {
	start := time.Now()

	if c.manager == nil {
		return CheckResult{
			Name:      c.Name(),
			Status:    StatusUnhealthy,
			Error:     "Localization manager is not initialized",
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}

	status := StatusHealthy
	message := "Localization system is healthy"

	// Get supported languages
	supportedLanguages := c.manager.GetSupportedLanguages()
	primaryLanguage := c.manager.GetPrimaryLanguage()
	languageCount := c.manager.GetLanguageCount()

	metadata := map[string]string{
		"supported_languages": fmt.Sprintf("%v", supportedLanguages),
		"primary_language":    primaryLanguage,
		"language_count":      fmt.Sprintf("%d", languageCount),
	}

	// Test key string retrieval for each supported language
	testKey := "welcome"
	workingLanguages := 0

	for _, lang := range supportedLanguages {
		testString := c.manager.GetString(testKey, lang)
		if testString != testKey { // If it returns the key, it means the translation was not found
			workingLanguages++
		}
	}

	metadata["working_languages"] = fmt.Sprintf("%d", workingLanguages)

	// Check if primary language is working
	primaryTest := c.manager.GetString(testKey, primaryLanguage)
	if primaryTest == testKey {
		status = StatusUnhealthy
		message = fmt.Sprintf("Primary language %s is not working properly", primaryLanguage)
	} else if workingLanguages < languageCount {
		status = StatusDegraded
		message = fmt.Sprintf("Some languages are not working: %d/%d functional", workingLanguages, languageCount)
	}

	// Test common keys
	commonKeys := []string{"welcome", "error", "help", "goodbye"}
	workingKeys := 0

	for _, key := range commonKeys {
		testString := c.manager.GetString(key, primaryLanguage)
		if testString != key {
			workingKeys++
		}
	}

	metadata["working_keys"] = fmt.Sprintf("%d/%d", workingKeys, len(commonKeys))

	if workingKeys == 0 {
		status = StatusUnhealthy
		message = "No localization keys are working"
	} else if workingKeys < len(commonKeys) {
		status = StatusDegraded
		message = fmt.Sprintf("Some localization keys are missing: %d/%d working", workingKeys, len(commonKeys))
	}

	return CheckResult{
		Name:      c.Name(),
		Status:    status,
		Message:   message,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
}

// HTTPServerHealthChecker checks HTTP server health
type HTTPServerHealthChecker struct {
	port string
}

func NewHTTPServerHealthChecker(port string) *HTTPServerHealthChecker {
	return &HTTPServerHealthChecker{port: port}
}

func (c *HTTPServerHealthChecker) Name() string {
	return "http_server"
}

func (c *HTTPServerHealthChecker) Check() CheckResult {
	start := time.Now()

	status := StatusHealthy
	message := "HTTP server is healthy"

	metadata := map[string]string{
		"port": c.port,
	}

	// Test local connectivity
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	url := fmt.Sprintf("http://localhost:%s/health", c.port)
	resp, err := client.Get(url)
	if err != nil {
		return CheckResult{
			Name:      c.Name(),
			Status:    StatusUnhealthy,
			Error:     fmt.Sprintf("Cannot connect to HTTP server: %v", err),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
			Metadata:  metadata,
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log error but don't fail the request
			_ = err // Suppress linter warning about empty branch
		}
	}()

	metadata["response_status"] = fmt.Sprintf("%d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		status = StatusDegraded
		message = fmt.Sprintf("HTTP server returned status %d", resp.StatusCode)
	}

	return CheckResult{
		Name:      c.Name(),
		Status:    status,
		Message:   message,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
}
