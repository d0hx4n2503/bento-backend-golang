package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics is the application-facing metrics surface.
type Metrics interface {
	IncHits(status int, method, route string)
	ObserveResponseTime(status int, method, route string, durationSeconds float64)
}

// PrometheusMetrics records HTTP request metrics in a private registry.
type PrometheusMetrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewPrometheusMetrics creates request metrics for one service. route must be
// a normalized route template (for example, "/users/:id"), never a raw URL.
func NewPrometheusMetrics(serviceName string) (*PrometheusMetrics, error) {
	registry := prometheus.NewRegistry()
	metrics := &PrometheusMetrics{
		registry: registry,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: serviceName,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests.",
		}, []string{"status", "method", "route"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: serviceName,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"status", "method", "route"}),
	}

	for _, collector := range []prometheus.Collector{
		metrics.requests,
		metrics.duration,
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	} {
		if err := registry.Register(collector); err != nil {
			return nil, err
		}
	}

	return metrics, nil
}

// Handler exposes this service's metrics registry. The application composition
// root owns the HTTP server and its startup and shutdown lifecycle.
func (m *PrometheusMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *PrometheusMetrics) IncHits(status int, method, route string) {
	m.requests.WithLabelValues(strconv.Itoa(status), method, route).Inc()
}

func (m *PrometheusMetrics) ObserveResponseTime(status int, method, route string, durationSeconds float64) {
	m.duration.WithLabelValues(strconv.Itoa(status), method, route).Observe(durationSeconds)
}
