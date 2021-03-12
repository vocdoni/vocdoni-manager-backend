package notify

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"go.vocdoni.io/dvote/chain"
	"go.vocdoni.io/dvote/chain/contracts"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/proto/build/go/models"
)

var ethereumEventList = map[string]string{
	// NewProcess(bytes32 processId, uint32 namespace)
	"processesNewProcess": "0x3b1cc0fc696cbe654bd83494847cc7890f2ae0e05a79dfbd6c1892061fbf3404",
}

// ProcessMeta returns the info of a newly created process from the event raised and ethereum storage
func ProcessMeta(ctx context.Context, contractABI *abi.ABI, eventData []byte, ph *chain.VotingHandle) (*models.NewProcessTx, error) {
	structuredData := &contracts.ProcessesNewProcess{}
	if err := contractABI.UnpackIntoInterface(structuredData, "NewProcess", eventData); err != nil {
		return nil, fmt.Errorf("cannot unpack NewProcess event: %w", err)
	}
	log.Debugf("newProcessMeta eventData: %+v", structuredData)
	return ph.NewProcessTxArgs(ctx, structuredData.ProcessId, structuredData.Namespace)
}

// @jordipainan TODO: func ResultsMeta()
