package registry

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.vocdoni.io/dvote/log"
)

// Registry collectors
var (
	// RegistryRequests ...
	RegistryRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "registry",
		Name:      "requests",
		Help:      "The number of registry requests",
	}, []string{"method"})
)

func (r *Registry) registerMetrics() {
	if r.ma != nil {
		log.Infof("registering metrics for registry")
		r.ma.Register(RegistryRequests)
	}
}
