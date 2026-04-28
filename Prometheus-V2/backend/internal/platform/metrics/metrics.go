package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Registry owns the Prometheus metrics for the V2 backend. The embedded
// *prometheus.Registry is the canonical place where domain modules register
// their own collectors via reg.MustRegister(...). The exported HTTP* fields
// are the platform-level metrics recorded by the http server middleware.
type Registry struct {
	*prometheus.Registry
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
}

// New builds a fresh registry with the platform-level collectors:
// Go runtime, process info, plus HTTP request count and latency histogram.
// Latency buckets cover roughly 5ms to 10s (exponential x2); tune per
// route family later if a slow background endpoint distorts SLOs.
func New() *Registry {
	r := prometheus.NewRegistry()
	r.MustRegister(collectors.NewGoCollector())
	r.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	httpReqs := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests labeled by method, route and numeric status code.",
	}, []string{"method", "route", "status"})
	r.MustRegister(httpReqs)

	httpDur := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Latency of HTTP requests labeled by method and route.",
		Buckets: prometheus.ExponentialBuckets(0.005, 2, 12),
	}, []string{"method", "route"})
	r.MustRegister(httpDur)

	return &Registry{
		Registry:            r,
		HTTPRequestsTotal:   httpReqs,
		HTTPRequestDuration: httpDur,
	}
}
