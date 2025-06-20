# Health Check System Documentation

## Overview

The OneBusAway Twilio health check system provides comprehensive monitoring and observability for production deployments. It implements industry-standard health check patterns with Kubernetes compatibility, Prometheus metrics, and robust error handling.

## Features

### Core Health Checks
- **System Health**: Memory, goroutines, GC monitoring
- **OneBusAway API**: API connectivity, response times, circuit breaker status
- **Session Store**: Cache performance, memory usage, cleanup status
- **Localization**: Language availability and string resolution
- **HTTP Server**: Self-connectivity verification

### Production Features
- **Kubernetes-ready endpoints**: `/health` (liveness), `/health/ready` (readiness)
- **Prometheus metrics**: Native format with proper labeling
- **Rate limiting**: 300 requests/minute per IP protection
- **Concurrent execution**: Semaphore-controlled parallelism
- **Intelligent caching**: 30-second TTL with automatic invalidation
- **Panic recovery**: All checks protected from runtime panics
- **Timeout handling**: Configurable timeouts with graceful degradation

## Quick Start

```go
// Initialize health manager
healthManager := health.NewManager(
    health.WithTimeout(10*time.Second),
    health.WithCacheTTL(30*time.Second),
    health.WithMaxConcurrentChecks(5),
    health.WithSystemInfo(true),
    health.WithMetrics(true),
)

// Register health checkers
healthManager.AddChecker(&health.SystemHealthChecker{})
healthManager.AddChecker(health.NewOneBusAwayHealthChecker(obaClient))
healthManager.AddChecker(health.NewSessionStoreHealthChecker(sessionStore))
healthManager.AddChecker(health.NewLocalizationHealthChecker(locManager))

// Create handler and setup routes
healthHandler := health.NewHandler(healthManager)
healthHandler.SetupRoutes(router)
```

## API Endpoints

### Core Endpoints

#### `GET /health` - Liveness Probe
**Purpose**: Kubernetes liveness probe  
**Timeout**: 5 seconds  
**Returns**: Basic system health status

```json
{
  "status": "healthy",
  "timestamp": "2025-06-20T11:16:03Z",
  "duration": "2.5ms",
  "checks": {
    "system": {
      "name": "system",
      "status": "healthy",
      "message": "System is healthy",
      "duration": "245μs",
      "timestamp": "2025-06-20T11:16:03Z",
      "metadata": {
        "goroutines": "25",
        "memory_alloc": "8.45 MB",
        "memory_sys": "15.32 MB",
        "gc_cycles": "12"
      }
    }
  }
}
```

#### `GET /health/ready` - Readiness Probe
**Purpose**: Kubernetes readiness probe  
**Timeout**: 10 seconds  
**Returns**: Comprehensive dependency health

#### `GET /health/detailed` - Comprehensive Status
**Purpose**: Debugging and monitoring  
**Timeout**: 15 seconds  
**Returns**: Full system status with dependencies

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2025-06-20T11:16:03Z",
  "duration": "15.2ms",
  "checks": {...},
  "system_info": {
    "go_version": "go1.21.0",
    "goroutines": 25,
    "memory": {
      "alloc": 8847360,
      "total_alloc": 12563840,
      "sys": 16056320,
      "num_gc": 12,
      "heap_alloc": 8847360,
      "heap_sys": 15499264,
      "heap_inuse": 10838016,
      "heap_released": 0,
      "usage_percent": 69.95
    },
    "uptime": "5m30s",
    "start_time": "2025-06-20T11:10:33Z",
    "healthy_checks": 4,
    "degraded_checks": 0,
    "failed_checks": 0
  },
  "dependencies": {
    "onebusaway_api": {
      "name": "OneBusAway API",
      "status": "healthy",
      "response_time": "150ms",
      "last_checked": "2025-06-20T11:16:03Z",
      "success_rate": 98.5,
      "error_count": 15,
      "request_count": 1000,
      "metadata": {
        "cache_hit_rate": 85.2,
        "circuit_breaker_opens": 0
      }
    }
  }
}
```

### Metrics and Monitoring

#### `GET /metrics` - Prometheus Metrics
**Purpose**: Prometheus scraping  
**Format**: Standard Prometheus text format

```
# HELP health_checks_total Total number of health checks performed
# TYPE health_checks_total counter
health_checks_total 1542.000000

# HELP health_checks_duration_seconds Average duration of health checks in seconds
# TYPE health_checks_duration_seconds gauge
health_checks_duration_seconds 0.015200

# HELP system_goroutines_total Current number of goroutines
# TYPE system_goroutines_total gauge
system_goroutines_total 25.000000

# HELP system_memory_alloc_bytes Current bytes allocated
# TYPE system_memory_alloc_bytes gauge
system_memory_alloc_bytes 8847360.000000
```

JSON format available with `Accept: application/json` header.

#### `GET /health/stats` - Basic Statistics
**Purpose**: Quick operational overview

```json
{
  "status": "ok",
  "stats": {
    "total_checks": 1542,
    "total_failures": 23,
    "average_duration": 0.0152,
    "success_rate": 98.51,
    "uptime": 330.5,
    "cache_size": 5,
    "checker_count": 5
  }
}
```

### Management Endpoints

#### `GET /health/config` - Configuration
**Purpose**: Runtime configuration inspection

```json
{
  "timeout": "10s",
  "cache_ttl": "30s",
  "max_concurrent_checks": 5,
  "system_info_enabled": true,
  "metrics_enabled": true,
  "registered_checkers": [
    "system",
    "onebusaway_api",
    "session_store",
    "localization",
    "http_server"
  ],
  "cache_size": 5
}
```

#### `GET /health/cache` - Cache Status
**Purpose**: Cache monitoring

```json
{
  "cache_size": 5,
  "cache_ttl": "30s"
}
```

#### `DELETE /health/cache` - Clear Cache
**Purpose**: Force cache invalidation

```json
{
  "status": "ok",
  "message": "Health check cache cleared"
}
```

## Health Status Levels

### `healthy`
- All systems operational
- Performance within acceptable ranges
- No critical errors detected

### `degraded`
- System functional but performance impacted
- Non-critical errors present
- Service remains available

### `unhealthy`
- Critical system failure
- Service unavailable
- Immediate attention required

## Configuration Options

```go
healthManager := health.NewManager(
    // Check timeout (default: 5s)
    health.WithTimeout(10*time.Second),
    
    // Cache TTL (default: 30s) 
    health.WithCacheTTL(60*time.Second),
    
    // Max concurrent checks (default: 10)
    health.WithMaxConcurrentChecks(5),
    
    // Enable system info collection (default: true)
    health.WithSystemInfo(true),
    
    // Enable metrics collection (default: true)
    health.WithMetrics(true),
)
```

## Custom Health Checkers

Implement the `HealthChecker` interface:

```go
type CustomChecker struct {
    service *MyService
}

func (c *CustomChecker) Name() string {
    return "my_service"
}

func (c *CustomChecker) Check() health.CheckResult {
    start := time.Now()
    
    // Perform health check logic
    status := health.StatusHealthy
    message := "Service is healthy"
    
    if err := c.service.Ping(); err != nil {
        status = health.StatusUnhealthy
        message = fmt.Sprintf("Service error: %v", err)
    }
    
    return health.CheckResult{
        Name:      c.Name(),
        Status:    status,
        Message:   message,
        Duration:  time.Since(start),
        Timestamp: time.Now(),
        Metadata: map[string]string{
            "version": c.service.Version(),
        },
    }
}

// Register the checker
healthManager.AddChecker(&CustomChecker{service: myService})
```

## Performance Characteristics

Based on benchmark testing:

| Component | Average Latency | Memory | Allocations |
|-----------|----------------|---------|-------------|
| SystemHealthChecker | 23.8μs | 400B | 9 |
| OneBusAwayHealthChecker | 1.13μs | 985B | 21 |
| SessionStoreHealthChecker | 401ns | 404B | 4 |
| MetricsUpdate | 52ns | 0B | 0 |

## Rate Limiting

Health endpoints are protected by rate limiting:
- **Limit**: 300 requests per minute per IP
- **Response**: HTTP 429 with `Retry-After` header
- **Cleanup**: Automatic expired entry removal

## Kubernetes Integration

### Liveness Probe
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

### Readiness Probe
```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 10
  failureThreshold: 2
```

### Prometheus Scraping
```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/path: "/metrics"
  prometheus.io/port: "8080"
```

## Monitoring and Alerting

### Key Metrics to Monitor
- `health_checks_failures_total` - Health check failure count
- `health_checks_duration_seconds` - Health check latency
- `system_memory_usage_percent` - Memory utilization
- `system_goroutines_total` - Goroutine leaks
- `dependency_response_time_seconds` - External service latency

### Alert Thresholds
- **Critical**: Any check status = `unhealthy`
- **Warning**: Any check status = `degraded`
- **Memory**: `system_memory_usage_percent > 90`
- **Goroutines**: `system_goroutines_total > 10000`
- **Latency**: `health_checks_duration_seconds > 1.0`

## Security Considerations

- **Rate limiting**: Prevents DoS attacks on health endpoints
- **Information disclosure**: Metadata filtered to prevent sensitive data exposure
- **Error sanitization**: Generic error messages for external consumption
- **Access control**: Consider adding authentication for detailed endpoints in production

## Troubleshooting

### Common Issues

**High memory usage**
```bash
curl http://localhost:8080/health/detailed | jq '.system_info.memory'
```

**Slow health checks**
```bash
curl http://localhost:8080/health/stats | jq '.stats.average_duration'
```

**Cache issues**
```bash
# Clear cache
curl -X DELETE http://localhost:8080/health/cache

# Check cache status
curl http://localhost:8080/health/cache
```

**Dependency failures**
```bash
curl http://localhost:8080/health/detailed | jq '.dependencies'
```

### Debug Mode
Set `GIN_MODE=debug` to enable detailed request logging.

## Contributing

When adding new health checkers:
1. Implement the `HealthChecker` interface
2. Add comprehensive tests with mocks
3. Include performance benchmarks
4. Document metadata fields
5. Follow existing patterns for error handling and timeouts