package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Registry struct {
	*prometheus.Registry
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
}

func New() *Registry {
	r := prometheus.NewRegistry()
	r.MustRegister(collectors.NewGoCollector())
	r.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	httpReqs := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests labeled by method, route and status code.",
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
