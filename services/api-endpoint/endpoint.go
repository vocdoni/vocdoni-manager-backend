package endpoint

import (
	"fmt"
	"time"

	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/metrics"
	"go.vocdoni.io/dvote/multirpc/transports"
	"go.vocdoni.io/dvote/multirpc/transports/mhttp"
	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/router"
)

// EndPoint handles the Websocket connection
type EndPoint struct {
	Router       *router.Router
	Proxy        *mhttp.Proxy
	MetricsAgent *metrics.Agent
}

// NewEndpoint creates a new websockets endpoint
func NewEndpoint(cfg *config.Manager, signer *ethereum.SignKeys) (*EndPoint, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cannot create endpoint, configuration is nil")
	}
	log.Infof("creating API service")
	pxy, err := proxy(cfg.API.ListenHost, cfg.API.ListenPort, cfg.API.Ssl.Domain, cfg.API.Ssl.DirCert)
	if err != nil {
		return nil, err
	}
	ts := new(mhttp.HttpWsHandler)
	ts.Init(new(transports.Connection))
	ts.SetProxy(pxy)

	listenerOutput := make(chan transports.Message)
	go ts.Listen(listenerOutput)
	transportMap := make(map[string]transports.Transport)
	transportMap["httpws"] = ts
	r := router.InitRouter(listenerOutput, transportMap, signer)
	var ma *metrics.Agent
	if cfg.Metrics != nil && cfg.Metrics.Enabled {
		ma = metrics.NewAgent("/metrics", time.Second*time.Duration(cfg.Metrics.RefreshInterval), pxy)
	}
	return &EndPoint{Router: r, Proxy: pxy, MetricsAgent: ma}, nil
}

// proxy creates a new service for routing HTTP connections using go-chi server
// if tlsDomain is specified, it will use letsencrypt to fetch a valid TLS certificate
func proxy(host string, port int, tlsDomain, tlsDir string) (*mhttp.Proxy, error) {
	pxy := mhttp.NewProxy()
	pxy.Conn.TLSdomain = tlsDomain
	pxy.Conn.TLScertDir = tlsDir
	pxy.Conn.Address = host
	pxy.Conn.Port = int32(port)
	log.Infof("creating proxy service, listening on %s:%d", host, port)
	if pxy.Conn.TLSdomain != "" {
		log.Infof("configuring proxy with TLS domain %s", tlsDomain)
	}
	return pxy, pxy.Init()
}
