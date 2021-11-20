package urlapi

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Router collectors
var (
	// RouterPrivateReqs ...
	RouterPrivateReqs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "router",
		Name:      "private_reqs",
		Help:      "The number of private requests processed",
	}, []string{"method"})
	// RouterPublicReqs ...
	RouterPublicReqs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "router",
		Name:      "public_reqs",
		Help:      "The number of public requests processed",
	}, []string{"method"})
)

func (a *URLAPI) registerMetrics() {
	if a.metricsagent == nil {
		return
	}
	a.metricsagent.Register(RouterPrivateReqs)
	a.metricsagent.Register(RouterPublicReqs)
}
