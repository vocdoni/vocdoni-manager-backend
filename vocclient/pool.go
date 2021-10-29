package vocclient

import (
	"fmt"

	"go.vocdoni.io/dvote/api"
	"go.vocdoni.io/dvote/client"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
)

// gateway client wrapper
type Gateway struct {
	client        *client.Client
	health        int32
	supportedApis []string
}

// list of clients, enables sorting by health
type GatewayPool []Gateway

func (pool GatewayPool) Len() int           { return len(pool) }
func (pool GatewayPool) Less(i, j int) bool { return pool[i].health > pool[j].health }
func (pool GatewayPool) Swap(i, j int)      { pool[i], pool[j] = pool[j], pool[i] }

func (pool GatewayPool) activeGateway() (Gateway, error) {
	if len(pool) == 0 {
		return Gateway{}, fmt.Errorf("no gateways available")
	}
	return (pool)[0], nil
}

func (pool *GatewayPool) shift() {
	log.Info(*pool)
	if len(*pool) < 2 {
		return
	}
	*pool = append((*pool)[1:], (*pool)[0])
	log.Info(*pool)
}

func (pool *GatewayPool) Request(req api.MetaRequest, signer *ethereum.SignKeys) (resp *api.MetaResponse, err error) {
	errorCount := 0
	// allow for 10 retries, shifting gateways each time
	for errorCount < 10 {
		gw, err := pool.activeGateway()
		if err != nil {
			return nil, fmt.Errorf("could not make request %s: %v", req.Method, err)
		}
		if !resp.Ok {
			return nil, fmt.Errorf("%s failed: %s", req.Method, resp.Message)
		}
		resp, err = gw.client.Request(req, signer)
		if err == nil {
			return resp, nil
		} else {
			errorCount++
			pool.shift()
		}
	}
	return
}
