package vocclient

import (
	"go.vocdoni.io/dvote/client"
)

type VocClient struct {
	pool       []*client.Client
	signingKey string
}

func New(gatewayUrls []string, signingKey string) (*VocClient, error) {

	gwPool, err := discoverGateways(gatewayUrls)
	if err != nil {
		return nil, err
	}
	return &VocClient{
		pool:       gwPool,
		signingKey: signingKey,
	}, nil
}
