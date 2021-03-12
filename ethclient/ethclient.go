package ethclient

import (
	"context"
	"fmt"
	"math/big"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.vocdoni.io/dvote/chain"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/types"
)

type Eth struct {
	networkName         string
	networkID           *big.Int
	provider            string
	gasLimit            uint64
	defaultFaucetAmount *big.Int
	client              *ethclient.Client
	signer              *ethereum.SignKeys
	timeout             time.Duration
}

// New creates a new SMTP object initialized with the user config
func New(ctx context.Context, ethc *config.EthNetwork, signer *ethereum.SignKeys) (*Eth, error) {
	var faucetAmount *big.Int

	// Get chain specs
	chainSpecs, err := chain.SpecsFor(ethc.Name)
	if err != nil {
		return nil, err
	}

	// Assign default values where needed
	provider := ethc.Provider
	if len(ethc.Provider) == 0 {
		return nil, fmt.Errorf("invalid ethereum provider")
	}
	gasLimit := ethc.GasLimit
	if gasLimit == 0 {
		gasLimit = types.DefaultGasLimit
	}

	if ethc.FaucetAmount > 0 {
		faucetAmount = big.NewInt(int64(ethc.FaucetAmount) * types.Finney)
	} else {
		defaultFaucetAmount, err := types.DefaultFaucetAmount(ethc.Name)
		if err != nil {
			return nil, err
		}
		faucetAmount = big.NewInt(int64(defaultFaucetAmount))
	}

	// Intstantiate Ethereum client
	ethclient, err := ethclient.Dial(provider)
	timeout := time.Duration(ethc.Timeout) * time.Second
	if err != nil {
		return nil, fmt.Errorf("cannot connect to ethereum endpoint: %w", err)
	}

	// Verify network ID
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	chainID, err := ethclient.NetworkID(tctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get network id: %s", err)
	}
	if chainID.Int64() != int64(chainSpecs.NetworkId) {
		return nil, fmt.Errorf("mismatch between Ethereum network name and ethereum network ID")
	}

	log.Debugf("%s", faucetAmount.String())
	log.Debugf("%s", timeout.String())

	return &Eth{client: ethclient, signer: signer, networkName: ethc.Name, networkID: chainID, provider: provider, gasLimit: gasLimit, defaultFaucetAmount: faucetAmount, timeout: timeout}, nil
}

func (eth *Eth) Close() {
	eth.client.Close()
}

func (eth *Eth) BalanceAt(ctx context.Context, address ethcommon.Address, blockNumber *big.Int) (*big.Int, error) {
	tctx, cancel := context.WithTimeout(ctx, eth.timeout)
	defer cancel()
	return eth.client.BalanceAt(tctx, address, blockNumber) // nil means latest block
}

// SendTokens sends gas to an address
// if the destination address has balance higher than maxAcceptedBalance the gas is not sent
// if the amount provided is 0 the the default amount of gas is used
func (eth *Eth) SendTokens(ctx context.Context, to ethcommon.Address, maxAcceptedBalance int64, amount int64) error {
	if eth.client == nil {
		return fmt.Errorf("cannot send tokens, ethereum client is nil")
	}

	// Check to address does not exceed maxAcceptedBalance
	tctx, cancel := context.WithTimeout(ctx, eth.timeout)
	defer cancel()
	toBalance, err := eth.BalanceAt(tctx, to, nil) // nil means latest block
	if err != nil {
		return fmt.Errorf("cannot check entity balance")
	}

	if toBalance.CmpAbs(big.NewInt(maxAcceptedBalance)) == 1 {
		return fmt.Errorf("not sending tokens, entity %s has already a balance of : %d", eth.signer.Address().String(), toBalance.Int64())
	}

	// Check manager has enough balance for the transfer
	tctx1, cancel1 := context.WithTimeout(ctx, eth.timeout)
	defer cancel1()
	fromBalance, err := eth.BalanceAt(tctx1, eth.signer.Address(), nil) // nil means latest block
	if err != nil {
		return fmt.Errorf("cannot check manager balance")
	}

	var value *big.Int
	if amount == 0 {
		value = eth.defaultFaucetAmount
	} else {
		value = big.NewInt(amount)
	}

	if fromBalance.CmpAbs(value) == -1 {
		return fmt.Errorf("cannot send tokens, wallet does not have enough balance: %d", fromBalance.Int64())
	}

	// set gas price
	var gasPrice *big.Int
	switch eth.networkName {
	// if xdai or sokol always 1 gwei
	case "xdai", "sokol":
		gasPrice = big.NewInt(1000000000) // 1 gwei
	// else let the node suggest
	default:
		tctx2, cancel2 := context.WithTimeout(ctx, eth.timeout)
		defer cancel2()
		gasPrice, err = eth.client.SuggestGasPrice(tctx2)
		if err != nil {
			return fmt.Errorf("cannot suggest gas price: %s", err)
		}
	}
	// get nonce for the signer
	tctx3, cancel3 := context.WithTimeout(ctx, eth.timeout)
	defer cancel3()
	nonce, err := eth.client.PendingNonceAt(tctx3, eth.signer.Address())
	if err != nil {
		return fmt.Errorf("cannot get signer account nonce: %s", err)
	}

	// create tx
	tx := ethtypes.NewTransaction(nonce, to, value, eth.gasLimit, gasPrice, []byte{})
	// sign tx
	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(eth.networkID), &eth.signer.Private)
	if err != nil {
		return fmt.Errorf("cannot sign transaction: %s", err)
	}
	// send tx
	tctx4, cancel4 := context.WithTimeout(ctx, eth.timeout)
	defer cancel4()
	err = eth.client.SendTransaction(tctx4, signedTx)
	if err != nil {
		return fmt.Errorf("cannot send signed tx: %s", err)
	}
	log.Infof("send %d tokens to newly created entity %s. TxHash: %s", value, to.String(), signedTx.Hash().Hex())
	return nil
}
