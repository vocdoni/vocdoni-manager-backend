package endpoint

import (
	"fmt"
	"time"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/net"
	"gitlab.com/vocdoni/go-dvote/types"
	"gitlab.com/vocdoni/manager/manager-backend/config"
	"gitlab.com/vocdoni/manager/manager-backend/router"
	"gitlab.com/vocdoni/manager/manager-backend/services/metrics"
)

// EndPoint handles the Websocket connection
type EndPoint struct {
	Router       *router.Router
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
	ws := new(net.WebsocketHandle)
	ws.Init(new(types.Connection))
	ws.SetProxy(pxy)
	listenerOutput := make(chan types.Message)
	go ws.Listen(listenerOutput)
	r := router.InitRouter(listenerOutput, ws, signer)
	var ma *metrics.Agent
	if cfg.Metrics != nil && cfg.Metrics.Enabled {
		ma = metrics.NewAgent("/metrics", time.Second*time.Duration(cfg.Metrics.RefreshInterval), pxy)
	}
	return &EndPoint{Router: r, MetricsAgent: ma}, nil
}

// proxy creates a new service for routing HTTP connections using go-chi server
// if tlsDomain is specified, it will use letsencrypt to fetch a valid TLS certificate
func proxy(host string, port int, tlsDomain, tlsDir string) (*net.Proxy, error) {
	pxy := net.NewProxy()
	pxy.C.SSLDomain = tlsDomain
	pxy.C.SSLCertDir = tlsDir
	pxy.C.Address = host
	pxy.C.Port = port
	log.Infof("creating proxy service, listening on %s:%d", host, port)
	if pxy.C.SSLDomain != "" {
		log.Infof("configuring proxy with TLS domain %s", tlsDomain)
	}
	return pxy, pxy.Init()
}
