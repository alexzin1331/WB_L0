package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	mu             sync.Mutex
	requests       map[string]int64
	durationTotals map[string]float64
}

func NewMetrics() *Metrics {
	return &Metrics{
		requests:       make(map[string]int64),
		durationTotals: make(map[string]float64),
	}
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		status := c.Writer.Status()
		key := metricKey(c.Request.Method, path, status)

		m.mu.Lock()
		m.requests[key]++
		m.durationTotals[key] += time.Since(start).Seconds()
		m.mu.Unlock()
	}
}

func (m *Metrics) Handler(c *gin.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	keys := make([]string, 0, len(m.requests))
	for key := range m.requests {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	builder.WriteString("# HELP wb_http_requests_total Total HTTP requests.\n")
	builder.WriteString("# TYPE wb_http_requests_total counter\n")
	for _, key := range keys {
		method, path, status := splitMetricKey(key)
		builder.WriteString(fmt.Sprintf(
			"wb_http_requests_total{method=%q,path=%q,status=%q} %d\n",
			method,
			path,
			status,
			m.requests[key],
		))
	}

	builder.WriteString("# HELP wb_http_request_duration_seconds_sum Total HTTP request duration in seconds.\n")
	builder.WriteString("# TYPE wb_http_request_duration_seconds_sum counter\n")
	for _, key := range keys {
		method, path, status := splitMetricKey(key)
		builder.WriteString(fmt.Sprintf(
			"wb_http_request_duration_seconds_sum{method=%q,path=%q,status=%q} %.6f\n",
			method,
			path,
			status,
			m.durationTotals[key],
		))
	}

	c.Data(http.StatusOK, "text/plain; version=0.0.4", []byte(builder.String()))
}

func metricKey(method, path string, status int) string {
	return fmt.Sprintf("%s\x00%s\x00%d", method, path, status)
}

func splitMetricKey(key string) (string, string, string) {
	parts := strings.Split(key, "\x00")
	if len(parts) != 3 {
		return "unknown", "unknown", "0"
	}
	return parts[0], parts[1], parts[2]
}
