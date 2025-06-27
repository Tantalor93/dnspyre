package dnsbench

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	dnsRequestsDurationMetrics = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "dnspyre",
		Name:      "dns_requests_duration_seconds",
		Help:      "DNS request duration in seconds",
	}, []string{"type"})

	dnsResponseTotalMetrics = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "dnspyre",
		Name:      "dns_response_total",
		Help:      "The total number DNS responses",
	}, []string{"type", "rcode"})

	errorsTotalMetrics = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "dnspyre",
		Name:      "errors_total",
		Help:      "The total number errors",
	}, []string{})
)
