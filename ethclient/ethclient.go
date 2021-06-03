package ethclient

import (
	"context"
	"fmt"
	"math/big"
	"time"

	goethereum "github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.vocdoni.io/dvote/chain"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/types"
)

type Signer struct {
	SignKeys *ethereum.SignKeys
	Taken    chan bool
}

func checkTxStatus(
	txHash *ethcommon.Hash,
	ethclient *ethclient.Client,
	timeout time.Duration) (uint64, error) {
	tctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	receipt, err := ethclient.TransactionReceipt(tctx, *txHash)
	if err != nil {
		return 0, err
	}
	return receipt.Status, nil
}

// send tokens and returns the hash of the tx
func (s *Signer) sendTokens(ctx context.Context,
	networkName string,
	ethclient *ethclient.Client,
	timeout time.Duration,
	gasLimit uint64,
	to ethcommon.Address,
	amount *big.Int) (*ethcommon.Hash, uint64, error) {
	// set gas price
	var err error
	var gasPrice = big.NewInt(60000000000) // 60 gwei
	switch networkName {
	case "sokol":
		gasPrice = big.NewInt(1000000000) // 10 gwei
	default:
		tctx2, cancel2 := context.WithTimeout(ctx, timeout)
		defer cancel2()
		gasPrice, err = ethclient.SuggestGasPrice(tctx2)
		if err != nil {
			log.Warn("Could not estimate gas price, using default value of 60gwei")
		}
	}
	// get nonce for the signer
	tctx2, cancel2 := context.WithTimeout(ctx, timeout)
	defer cancel2()
	nonce, err := ethclient.PendingNonceAt(tctx2, s.SignKeys.Address())
	if err != nil {
		return nil, 0, fmt.Errorf("cannot get signer account nonce: %s", err)
	}
	// create tx
	tx := ethtypes.NewTransaction(nonce, to, amount, gasLimit, gasPrice, nil)
	// sign tx
	tctx3, cancel3 := context.WithTimeout(ctx, timeout)
	defer cancel3()
	networkId, err := ethclient.NetworkID(tctx3)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot get networkId: %w", err)
	}
	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(networkId), &s.SignKeys.Private)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot sign transaction: %s", err)
	}
	// send tx
	tctx4, cancel4 := context.WithTimeout(ctx, timeout)
	defer cancel4()
	err = ethclient.SendTransaction(tctx4, signedTx)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot send signed tx: %s", err)
	}
	log.Infof("sending %d tokens to newly created entity %s from signer: %s. TxHash: %s and Nonce: %d",
		amount,
		to.String(),
		s.SignKeys.AddressString(),
		signedTx.Hash().Hex(),
		signedTx.Nonce(),
	)
	nHash := new(ethcommon.Hash)
	*nHash = signedTx.Hash()
	return nHash, signedTx.Nonce(), nil
}

func (s *Signer) checkEnoughBalance(ctx context.Context,
	defaultAmount *big.Int,
	ethclient *ethclient.Client,
	timeout time.Duration) (bool, error) {
	// Check manager has enough balance for the transfer
	tctx1, cancel1 := context.WithTimeout(ctx, timeout)
	defer cancel1()
	fromBalance, err := ethclient.BalanceAt(tctx1, s.SignKeys.Address(), nil) // nil means latest block
	if err != nil {
		return false, fmt.Errorf("cannot check manager balance")
	}
	var value *big.Int
	var amount int64
	if amount == 0 {
		value = defaultAmount
	} else {
		value = big.NewInt(amount)
	}
	if fromBalance.CmpAbs(value) == -1 {
		return false, fmt.Errorf("wallet does not have enough balance: %d", fromBalance.Int64())
	}
	return true, nil
}

type Eth struct {
	networkName         string
	networkID           *big.Int
	provider            string
	gasLimit            uint64
	DefaultFaucetAmount *big.Int
	client              *ethclient.Client
	SignersPool         []*Signer
	timeout             time.Duration
}

// New creates a new Eth object initialized with the user config
func New(ctx context.Context, ethc *config.EthNetwork, signersPool []*Signer) (*Eth, error) {
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
	// Instantiate Ethereum client
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
	return &Eth{
		client:              ethclient,
		SignersPool:         signersPool,
		networkName:         ethc.Name,
		networkID:           chainID,
		provider:            provider,
		gasLimit:            gasLimit,
		DefaultFaucetAmount: faucetAmount,
		timeout:             timeout,
	}, nil
}

func (eth *Eth) Close() {
	eth.client.Close()
}

func (eth *Eth) BalanceAt(ctx context.Context,
	address ethcommon.Address,
	blockNumber *big.Int) (*big.Int, error) {
	tctx, cancel := context.WithTimeout(ctx, eth.timeout)
	defer cancel()
	return eth.client.BalanceAt(tctx, address, blockNumber) // nil means latest block
}

// SendTokens sends gas to an address
// if the destination address has balance higher than maxAcceptedBalance the gas is not sent
// if the amount provided is 0 the the default amount of gas is used
func (eth *Eth) SendTokens(ctx context.Context,
	to ethcommon.Address,
	maxAcceptedBalance int64,
	amount int64) (*big.Int, error) {
	sent := &big.Int{}
	if eth.client == nil {
		return sent, fmt.Errorf("cannot send tokens, ethereum client is nil")
	}
	// Check to address does not exceed maxAcceptedBalance
	tctx, cancel := context.WithTimeout(ctx, eth.timeout)
	defer cancel()
	toBalance, err := eth.BalanceAt(tctx, to, nil) // nil means latest block
	if err != nil {
		return sent, fmt.Errorf("cannot check entity balance")
	}
	if toBalance.CmpAbs(big.NewInt(maxAcceptedBalance)) == 1 {
		return sent, fmt.Errorf("entity %s has already a balance of : %d, greater than the maxAcceptedBalance",
			to.String(),
			toBalance.Int64(),
		)
	}
	finished := false
	// get available signer

	for {
		for _, signer := range eth.SignersPool {
			select {
			case signer.Taken <- true:
			default:
				log.Debugf("signer %s has a pending tx",
					signer.SignKeys.AddressString())
				continue
			}
			// check all signer pending txs
			log.Debugf("using signer %s", signer.SignKeys.AddressString())
			tctx2, cancel2 := context.WithTimeout(ctx, eth.timeout)
			defer cancel2()
			// if signer has not enough balance or error checking it select the next one
			isEnough, err := signer.checkEnoughBalance(tctx2, eth.DefaultFaucetAmount, eth.client, eth.timeout)
			if err != nil {
				log.Infof("cannot check signer: %s balance with error: %s", signer.SignKeys.Address().Hex(), err)
				<-signer.Taken
				continue
			}
			if !isEnough {
				log.Infof("signer %s have not enough balance", signer.SignKeys.Address().Hex())
				<-signer.Taken
				continue
			}
			// send tx
			tctx3, cancel3 := context.WithTimeout(ctx, eth.timeout)
			defer cancel3()
			var value *big.Int
			if amount == 0 {
				value = eth.DefaultFaucetAmount
			} else {
				value = big.NewInt(amount)
			}
			txHash, nonce, err := signer.sendTokens(tctx3,
				eth.networkName,
				eth.client,
				eth.timeout,
				eth.gasLimit,
				to,
				value,
			)
			if err != nil {
				log.Infof("cannot send tx: %s with signer: %s", txHash.Hex(), signer.SignKeys.Address().Hex())
				<-signer.Taken
				continue
			}
			// add pending tx
			log.Infof("signer %s txhash: %s with nonce: %d sended successfully",
				signer.SignKeys.Address().Hex(),
				txHash.String(),
				nonce,
			)
			log.Debugf("added pending tx to signer: %s", signer.SignKeys.AddressString())
			go signer.waitForTx(eth.client, eth.timeout*2, txHash)
			finished = true
			break
		}
		// wait for signers
		if finished {
			break
		}
		time.Sleep(time.Second * 5)
	}
	return eth.DefaultFaucetAmount, nil
}

func (s *Signer) waitForTx(ethclient *ethclient.Client,
	timeout time.Duration, txHash *ethcommon.Hash) {
	// try get transaction receipt
	// if not found wait
	// if not found after waiting free the signer
	log.Debugf("waiting tx for signer: %s", s.SignKeys.AddressString())
	var status uint64
	var err error
	for {
		status, err = checkTxStatus(txHash, ethclient, timeout)
		if err != nil {
			if err == goethereum.NotFound {
				// TODO: find a better way than polling
				time.Sleep(time.Second * 5) // wait before checking again
				continue
			} else {
				log.Warnf("cannot check signer: %s tx hash %s status with err: %s",
					s.SignKeys.Address().Hex(),
					txHash.Hex(),
					err,
				)
				break
			}
		} else {
			log.Debugf("tx %s status is: %d", txHash.Hex(), status)
			if status == 0 {
				log.Warnf("signer %s tx %s failed on execution", s.SignKeys.Address().Hex(), txHash.Hex())
			} else {
				log.Infof("signer %s tx %s succesfully executed", s.SignKeys.Address().Hex(), txHash.Hex())
			}
			break
		}
	}
	<-s.Taken
}
