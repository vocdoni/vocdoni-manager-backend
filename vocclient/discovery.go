package vocclient

import (
	"fmt"
	"sort"
	"sync"

	"go.vocdoni.io/dvote/api"
	"go.vocdoni.io/dvote/client"
	"go.vocdoni.io/dvote/log"
)

// client wrapper for sorting by health
type sortableClient struct {
	client *client.Client
	health int32
}

// list of clients for sorting by health
type sortableClients []sortableClient

func (c sortableClients) Len() int           { return len(c) }
func (c sortableClients) Less(i, j int) bool { return c[i].health > c[j].health }
func (c sortableClients) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

func discoverGateways(urls []string) ([]*client.Client, error) {
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
		return gateways, fmt.Errorf("could not initialize %d gateway clients", len(urls))
	}
	sortGateways(gateways)
	log.Debugf("successfully connected to %d gateways", len(gateways))
	return gateways, nil
}

func sortGateways(gateways []*client.Client) {
	// make list of gateways that can be sorted by health
	sortableGateways := sortableClients{}
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
				sortableGateways = append(
					sortableGateways, sortableClient{
						client: nil,
						health: 0,
					})
			} else {
				sortableGateways = append(
					sortableGateways, sortableClient{
						client: gw,
						health: resp.Health,
					})
			}
			mtx.Unlock()
			wg.Done()
		}(gateway, i)
	}
	wg.Wait()

	// sort gateways according to health scores
	sort.Stable(sortableGateways)

	// assign original gateway list in order of health
	for i, gateway := range sortableGateways {
		gateways[i] = gateway.client
	}
}
