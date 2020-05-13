package endpoint

import (
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/net"
	"gitlab.com/vocdoni/go-dvote/types"
	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
)

// EndPoint handles the Websocket connection
type EndPoint struct {
	WS *net.WebsocketHandle
}

// NewEndpoint creates a new websockets endpoint
func NewEndpoint(cfg *config.Manager) (*EndPoint, error) {
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

	return &EndPoint{WS: ws}, nil
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
