package vocclient

import (
	"fmt"
	"sort"
	"sync"

	"go.vocdoni.io/dvote/api"
	"go.vocdoni.io/dvote/client"
	"go.vocdoni.io/dvote/log"
)

func discoverGateways(urls []string) (GatewayPool, error) {
	gateways := []*client.Client{}
	log.Debugf("discovering gateways %v", urls)
	for _, url := range urls {
		client, err := client.New(url)
		if err != nil {
			log.Warnf("Could not connect to gateway %s: %v", url, err)
		} else {
			gateways = append(gateways, client)
		}
	}
	if len(gateways) == 0 {
		return nil, fmt.Errorf("could not initialize %d gateway clients", len(urls))
	}
	return sortGateways(gateways), nil
}

func sortGateways(gateways []*client.Client) GatewayPool {
	// make list of gateways that can be sorted by health
	gatewayPool := GatewayPool{}
	wg := &sync.WaitGroup{}
	mtx := new(sync.Mutex)

	// fetch & record the health of each gateway
	for i, gateway := range gateways {
		wg.Add(1)
		go func(gw *client.Client, index int) {
			resp, err := gw.Request(api.MetaRequest{Method: "getInfo"}, nil)
			mtx.Lock()
			if err != nil {
				log.Warnf("could not get info for %s: %v", gw.Addr, err)
				gatewayPool = append(
					gatewayPool, Gateway{
						client: nil,
						health: 0,
					})
			} else {
				gatewayPool = append(
					gatewayPool, Gateway{
						client:        gw,
						health:        resp.Health,
						supportedApis: resp.APIList,
					})
			}
			mtx.Unlock()
			wg.Done()
		}(gateway, i)
	}
	wg.Wait()

	// sort gateways according to health scores
	sort.Stable(gatewayPool)
	log.Debugf("successfully connected to %d gateways", len(gateways))
	return gatewayPool
}
