package testsmtp

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/ethclient"
)

var signer *ethereum.SignKeys

var testNetworks = []config.EthNetwork{
	{Name: "xdai", Provider: "https://xdai1.vocdoni.net", Timeout: 60},
	{Name: "goerli", Provider: "https://goerli.eth.aragon.network", Timeout: 60},
}

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	signer = ethereum.NewSignKeys()
	err := signer.Generate()
	if err != nil {
		fmt.Printf("Error initializiting ethereum signer: %v", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestConnect(t *testing.T) {
	for _, ethc := range testNetworks {
		t.Run(fmt.Sprintf("type=%s", ethc.Name), func(t *testing.T) {
			signers := make([]*ethclient.Signer, 1)
			signers[0] = &ethclient.Signer{SignKeys: signer}
			e, err := ethclient.New(context.Background(), &ethc, signers)
			if err != nil {
				t.Fatalf("unable to connect to default %s provider: (%v)", ethc.Name, err)
			}
			e.Close()
		})
	}
}

func TestBalanceAt(t *testing.T) {
	for _, ethc := range testNetworks {
		t.Run(fmt.Sprintf("type=%s", ethc.Name), func(t *testing.T) {
			signers := make([]*ethclient.Signer, 1)
			signers[0] = &ethclient.Signer{SignKeys: signer}
			e, err := ethclient.New(context.Background(), &ethc, signers)
			if err != nil {
				t.Fatalf("unable to connect to default %s provider: (%v)", ethc.Name, err)
			}
			balance, err := e.BalanceAt(context.Background(), signer.Address(), nil)
			qt.Assert(t, err, qt.IsNil)
			qt.Assert(t, balance.Int64(), qt.Equals, int64(0))
			e.Close()
		})
	}
}
