package urlapi

import (
	"fmt"
	"strings"

	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/metrics"
	"go.vocdoni.io/manager/manager"
	"go.vocdoni.io/manager/notify"
	"go.vocdoni.io/manager/registry"
)

type URLAPI struct {
	PrivateCalls uint64
	PublicCalls  uint64
	BaseRoute    string

	router       *httprouter.HTTProuter
	api          *bearerstdapi.BearerStandardAPI
	metricsagent *metrics.Agent
	manager      *manager.Manager
	registry     *registry.Registry
	notif        *notify.API
}

func NewURLAPI(router *httprouter.HTTProuter, baseRoute string, metricsAgent *metrics.Agent) (*URLAPI, error) {
	if router == nil {
		return nil, fmt.Errorf("httprouter is nil")
	}
	if len(baseRoute) == 0 || baseRoute[0] != '/' {
		return nil, fmt.Errorf("invalid base route (%s), it must start with /", baseRoute)
	}
	// Remove trailing slash
	if len(baseRoute) > 1 {
		baseRoute = strings.TrimSuffix(baseRoute, "/")
	}
	urlapi := URLAPI{
		BaseRoute:    baseRoute,
		router:       router,
		metricsagent: metricsAgent,
	}
	urlapi.registerMetrics()
	var err error
	urlapi.api, err = bearerstdapi.NewBearerStandardAPI(router, baseRoute)
	if err != nil {
		return nil, err
	}

	return &urlapi, nil
}
