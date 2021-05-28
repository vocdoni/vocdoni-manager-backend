package ethclient

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	goethhex "github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
	"go.vocdoni.io/dvote/chain"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/types"
)

// rpcTransaction wraps useful information about an EVM transaction
type rpcTransaction struct {
	BlockHash        *ethcommon.Hash    `json:"blockHash"`
	BlockNumber      *goethhex.Big      `json:"blockNumber"`
	From             ethcommon.Address  `json:"from"`
	Gas              goethhex.Uint64    `json:"gas"`
	GasPrice         *goethhex.Big      `json:"gasPrice"`
	Hash             ethcommon.Hash     `json:"hash"`
	Input            goethhex.Bytes     `json:"input"`
	Nonce            goethhex.Uint64    `json:"nonce"`
	To               *ethcommon.Address `json:"to"`
	TransactionIndex *goethhex.Uint64   `json:"transactionIndex"`
	Value            *goethhex.Big      `json:"value"`
	RpcType          goethhex.Uint64    `json:"type"`
	ChainID          *goethhex.Big      `json:"chainId,omitempty"`
	V                *goethhex.Big      `json:"v"`
	R                *goethhex.Big      `json:"r"`
	S                *goethhex.Big      `json:"s"`
}

// pendingTx wraps useful information for a pending transaction
type pendingTx rpcTransaction

// SignerWithPending wraps an Ethereum signer and a map
// for tracking pending transactions for that signer
type SignerWithPendingTx struct {
	// signKey signer for the transaction
	signKey *ethereum.SignKeys
	// pendingTx
	pendingTx *pendingTx
}

// SetKeyPair sets the ethereum signing key to the signer
func (s *SignerWithPendingTx) SetKeyPair(kp *ethereum.SignKeys) {
	s.signKey = kp
}

// Ethereum wraps the Ethereum RPC provider and Client
// amont other useful information for interacting with
// and underlying node
type Ethereum struct {
	// DialAddress endpoint URL to connect with
	DialAddress string
	// Client the Web3 client instance
	Client *ethclient.Client
	// RPC the RPC client instance
	RPC *ethrpc.Client
	// NetworkName name of the network
	NetworkName string
	// NetworkId identifies univocally the network
	NetworkID *big.Int
	// Timeout applied to ethereum transactions
	Timeout time.Duration
}

// Faucet wraps the required components for sending value to other accounts
type Faucet struct {
	// ethereum
	ethereum *Ethereum
	// signersWithPendingTxsPool list of signers with pending transactions
	signersWithPendingTxPool map[ethcommon.Address]*SignerWithPendingTx
	// amount amount to send by the faucet in Ether
	amount *big.Int
	// gasPrice gas price for sending a transaction
	gasPrice *big.Int
	// gasLimit gas limit for sending a transaction
	gasLimit int64
	// maxBalance is the maximum amount an address can have
	// in order to receive more faucet funds
	maxBalance *big.Int
}

// Geth tx_pool RPC call response required

// gethTxPoolResponseObject represents a transaction that will serialize
// to the RPC representation of a transaction
type gethTxPoolResponseObject = map[string]map[string]map[string]*rpcTransaction

// Parity parity_pendingTransactions RPC call response
type object = map[string]interface{}
type array = []interface{}

// NewEthereum creates a new Ethereum object initialized
// with the provided configuration
func NewEthereum(ctx context.Context,
	ethCfg *config.EthereumCfg) (*Ethereum, error) {

	eth := &Ethereum{}
	// Get chain specs
	chainSpecs, err := chain.SpecsFor(ethCfg.Name)
	if err != nil {
		return nil, err
	}

	// check dial address
	dialAddress := ethCfg.DialAddress
	if len(ethCfg.DialAddress) == 0 {
		return nil, fmt.Errorf("invalid ethereum provider")
	}
	eth.DialAddress = ethCfg.DialAddress
	maxtries := 10
	// try connect to the provided endpoint
	for {
		if maxtries == 0 {
			return nil, fmt.Errorf("could not connect to web3 endpoint")
		}
		// create RPC
		eth.RPC, err = ethrpc.Dial(dialAddress)
		if err != nil || eth.RPC == nil {
			log.Warnf("cannot create an ethereum rpc connection: (%s), trying again ...", err)
			time.Sleep(time.Second * 2)
			maxtries--
			continue
		}
		break
	}
	// if RPC connection established, create an ethereum client using the RPC client
	eth.Client = ethclient.NewClient(eth.RPC)
	// set global timeout for transactions
	eth.Timeout = time.Duration(ethCfg.Timeout) * time.Second

	// verify network ID
	tctx, cancel := context.WithTimeout(ctx, eth.Timeout)
	defer cancel()
	chainID, err := eth.Client.NetworkID(tctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get network id: %s", err)
	}
	if chainID.Int64() != int64(chainSpecs.NetworkId) {
		return nil, fmt.Errorf("mismatch between Ethereum network name and ethereum network ID")
	}

	return eth, nil
}

func (eth *Ethereum) balanceAt(ctx context.Context, address ethcommon.Address, blockNumber *big.Int) (*big.Int, error) {
	tctx, cancel := context.WithTimeout(ctx, eth.Timeout)
	defer cancel()
	// nil blockNumber means latest block
	return eth.Client.BalanceAt(tctx, address, blockNumber)
}

func NewFaucet(faucetCfg *config.FaucetCfg, eth *Ethereum) (*Faucet, error) {
	var err error
	// set gas limit
	gasLimit := faucetCfg.GasLimit
	if gasLimit == 0 {
		gasLimit = types.DefaultGasLimit
	}
	// set gas price
	gasPrice := faucetCfg.GasPrice
	if gasPrice == 0 {
		gasPrice, err = types.DefaultFaucetGasPrice(eth.NetworkName)
		if err != nil {
			return nil, fmt.Errorf("cannot set faucet gas price: %s", err)
		}
	}
	// set amount to send
	amount := faucetCfg.Amount
	if amount == 0 {
		amount, err = types.DefaultFaucetAmount(eth.NetworkName)
		if err != nil {
			return nil, fmt.Errorf("cannot set faucet gas limit: %s", err)
		}
	}
	// check ethereum client is set
	if eth.Client == nil || eth.RPC == nil {
		return nil, fmt.Errorf("Ethereum client is not connected")
	}
	// create signers with pending tx
	signers := make(map[ethcommon.Address]*SignerWithPendingTx, len(faucetCfg.Signers))
	for _, signerKey := range faucetCfg.Signers {
		kpair := ethereum.NewSignKeys()
		if err := kpair.AddHexKey(signerKey); err != nil {
			return nil, fmt.Errorf("cannot add signer key: %s", err)
		}
		signers[kpair.Address()] = &SignerWithPendingTx{
			signKey: kpair,
		}
	}
	return &Faucet{
		ethereum:                 eth,
		signersWithPendingTxPool: signers,
		amount:                   big.NewInt(amount),
		gasLimit:                 gasLimit,
		gasPrice:                 big.NewInt(gasPrice),
		maxBalance:               big.NewInt(faucetCfg.MaxBalance),
	}, nil
}

// checkEnoughtBalance checks if a signer have enough balance for sending a faucet tx
func (f *Faucet) checkEnoughBalance(ctx context.Context, signerAddr ethcommon.Address) (bool, error) {
	// Check manager has enough balance for the transfer
	tctx, cancel := context.WithTimeout(ctx, f.ethereum.Timeout)
	defer cancel()
	fromBalance, err := f.ethereum.Client.BalanceAt(tctx,
		signerAddr,
		nil) // nil means latest block
	if err != nil {
		return false, fmt.Errorf("cannot check signer %s balance", signerAddr.Hex())
	}
	if fromBalance.CmpAbs(f.amount) == -1 {
		return false, fmt.Errorf("wallet %s does not have enough balance: %d",
			signerAddr.Hex(),
			fromBalance.Int64())
	}
	return true, nil
}

// open/ethereumPendingMempoolTxs checks all pending txs of a signer
// looking into an OpenEthereum node mempool
func (f *Faucet) openEthereumPendingMempoolTxs(ctx context.Context,
	signerAddr ethcommon.Address) ([]*pendingTx, error) {
	tctx, cancel := context.WithTimeout(ctx, f.ethereum.Timeout)
	defer cancel()
	pendingTxs := make([]*pendingTx, 0)
	// create request
	// see: https://openethereum.github.io/JSONRPC-parity-module#parity_pendingtransactions
	resp := object{}
	params := array{}
	innerParams := object{}
	innerParams["limit"] = nil
	innerParams["filter"] = object{
		"limit": nil,
		"filter": object{
			"from": object{
				"eq": signerAddr.Hex(), // filter by address
			},
			"value": object{
				"eq": f.amount.Int64(), // filter by faucet amount
			},
		},
	}
	params[0] = innerParams
	// send RPC call
	if err := f.ethereum.RPC.CallContext(tctx,
		&resp,
		"parity_pendingTransactions",
		params...,
	); err != nil {
		return nil, fmt.Errorf("RPC call failed: %s", err)
	}
	// unwrap response required fields
	innerResultArray, _ := resp["result"].(array)
	for _, innerResult := range innerResultArray {
		// innerResult is a kv object
		innerResultMap, _ := innerResult.(object)
		// convert from interface
		from, _ := innerResultMap["from"].(string)
		to, _ := innerResultMap["to"].(string)
		value, _ := innerResultMap["value"].(string)
		valueBig, err := goethhex.DecodeBig(value)
		if err != nil {
			return nil, fmt.Errorf("cannot convert 'value' string to big int: %w", err)
		}
		hash, _ := innerResultMap["hash"].(string)
		nonce, _ := innerResultMap["nonce"].(string)
		nonceUint64, err := goethhex.DecodeUint64(nonce)
		if err != nil {
			return nil, fmt.Errorf("cannot convert 'nonce' string to uint64: %w", err)
		}
		gasPrice, _ := innerResultMap["gasPrice"].(string)
		gasPriceBig, err := goethhex.DecodeBig(gasPrice)
		if err != nil {
			return nil, fmt.Errorf("cannot convert 'gasPrice' string to big int: %w", err)
		}
		// format response data
		pendingTx := new(pendingTx)
		pendingTx.Nonce = goethhex.Uint64(nonceUint64)
		pendingTx.From = ethcommon.HexToAddress(from)
		toPtr := new(ethcommon.Address)
		*toPtr = ethcommon.HexToAddress(to)
		pendingTx.To = toPtr
		pendingTx.GasPrice = (*goethhex.Big)(gasPriceBig)
		pendingTx.Value = (*goethhex.Big)(valueBig)
		pendingTx.Hash = ethcommon.HexToHash(hash)
		// add pending tx to pending txs
		pendingTxs = append(pendingTxs, pendingTx)
	}
	return pendingTxs, nil
}

// gethPendingMempoolTxs checks all pending txs of a signer looking into
// the go-ethereum node mempool
func (f *Faucet) gethPendingMempoolTxs(ctx context.Context,
	signerAddr ethcommon.Address) ([]*pendingTx, error) {
	tctx, cancel := context.WithTimeout(ctx, f.ethereum.Timeout)
	defer cancel()
	pendingTxs := make([]*pendingTx, 0)

	var resp gethTxPoolResponseObject
	if err := f.ethereum.RPC.CallContext(tctx,
		&resp,
		"txpool_content",
		nil,
	); err != nil {
		return nil, fmt.Errorf("RPC call failed: %s", err)
	}
	// pending tx -> ready to be processed & included in a block
	// queued tx -> txs where the nonce is not in sequence
	// ignore queued tx
	pending := resp["pending"]
	for from, txs := range pending {
		// get all pending tx from signer
		if from != signerAddr.Hex() {
			continue
		}
		for _, tx := range txs {
			// get txs with the faucet value
			if tx.Value.ToInt() != f.amount {
				continue
			}
			// format response data
			pendingTx := new(pendingTx)
			pendingTx.Nonce = tx.Nonce
			pendingTx.From = tx.From
			pendingTx.To = tx.To
			pendingTx.GasPrice = tx.GasPrice
			pendingTx.Value = tx.Value
			pendingTx.Hash = tx.Hash
			// add pending tx to corresponding faucet signer pending txs
			pendingTxs = append(pendingTxs, pendingTx)
		}
	}
	return pendingTxs, nil
}

// send tokens and returns the transaction information
func (f *Faucet) sendTokens(ctx context.Context, signer *ethereum.SignKeys, to ethcommon.Address) (*ethtypes.Transaction, error) {
	// set gas price
	var err error
	switch f.ethereum.NetworkName {
	// if xdai or sokol always 1 gwei
	case "sokol", "xdai":
		break
	// else let the node suggest
	default:
		tctx, cancel := context.WithTimeout(ctx, f.ethereum.Timeout)
		defer cancel()
		f.gasPrice, err = f.ethereum.Client.SuggestGasPrice(tctx)
		if err != nil {
			log.Warn("Could not estimate gas price, using default value of 60gwei")
		}
	}
	// get nonce for the signer
	tctx2, cancel2 := context.WithTimeout(ctx, f.ethereum.Timeout)
	defer cancel2()
	nonce, err := f.ethereum.Client.PendingNonceAt(tctx2, signer.Address())
	if err != nil {
		return nil, fmt.Errorf("cannot get signer account nonce: %s", err)
	}
	// create tx
	tx := ethtypes.NewTransaction(nonce, to, f.amount, uint64(f.gasLimit), f.gasPrice, nil)
	// sign tx
	tctx3, cancel3 := context.WithTimeout(ctx, f.ethereum.Timeout)
	defer cancel3()
	networkId, err := f.ethereum.Client.NetworkID(tctx3)
	if err != nil {
		return nil, fmt.Errorf("cannot get networkId: %w", err)
	}
	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(networkId), &signer.Private)
	if err != nil {
		return nil, fmt.Errorf("cannot sign transaction: %s", err)
	}
	// send tx
	tctx4, cancel4 := context.WithTimeout(ctx, f.ethereum.Timeout)
	defer cancel4()
	log.Debugf("sending transaction %v", signedTx)
	err = f.ethereum.Client.SendTransaction(tctx4, signedTx)
	if err != nil {
		return nil, fmt.Errorf("cannot send signed tx: %s", err)
	}
	log.Infof("send %s tokens to %s with tx hash: %s",
		f.amount.String(),
		to.String(),
		signedTx.Hash().Hex())

	return signedTx, nil
}

// SendTokens sends gas to an address
// if the destination address has balance higher than maxBalance the gas is not sent
// if the amount provided is 0 the the default amount of gas is used
// The selected signer must not have any pending tx
func (f *Faucet) SendTokens(ctx context.Context, to ethcommon.Address) (*big.Int, error) {
	sent := new(big.Int)
	if f.ethereum.Client == nil {
		return nil, fmt.Errorf("cannot send tokens, ethereum client is nil")
	}
	// Check to address does not exceed maxBalance
	tctx, cancel := context.WithTimeout(ctx, f.ethereum.Timeout)
	defer cancel()
	toBalance, err := f.ethereum.balanceAt(tctx, to, nil) // nil means latest block
	if err != nil {
		return nil, fmt.Errorf("cannot check entity balance")
	}

	if toBalance.CmpAbs(f.maxBalance) == 1 {
		return nil, fmt.Errorf("entity %s has already a balance of : %s, greater than the maxBalance", to.String(), toBalance.String())
	}

	// get available signer
	for _, signer := range f.signersWithPendingTxPool {
		signerHexAddress := signer.signKey.Address().Hex()
		// check all signer pending txs
		tctx2, cancel2 := context.WithTimeout(ctx, f.ethereum.Timeout)
		defer cancel2()
		if signer.pendingTx != nil {
			// signer have pending tx, skip and use another signer
			continue

		}
		// if signer has not enough balance or error checking it select the next one
		signerEnoughBalance, err := f.checkEnoughBalance(tctx2, signer.signKey.Address())
		if err != nil {
			log.Warnf("cannot check signer: %s balance with error: %s", signerHexAddress, err)
			// if error checking enough balance, continue
			continue
		}
		// if signer does not have enough balance, get another one
		if !signerEnoughBalance {
			log.Warnf("signer %s have not enough balance", signerHexAddress)
			continue
		}
		// send tx
		tctx3, cancel3 := context.WithTimeout(ctx, f.ethereum.Timeout)
		defer cancel3()
		tx, err := f.sendTokens(tctx3, signer.signKey, to)
		if err != nil {
			log.Warnf("cannot send tx: %s with signer: %s", tx.Hash(), signerHexAddress)
			continue
		}
		// add pending tx
		pendingTx := new(pendingTx)
		pendingTx.Nonce = goethhex.Uint64(tx.Nonce())
		pendingTx.From = signer.signKey.Address()
		pendingTx.To = tx.To()
		pendingTx.GasPrice = (*goethhex.Big)(tx.GasPrice())
		pendingTx.Value = (*goethhex.Big)(tx.Value())
		signer.pendingTx = pendingTx
		sent = tx.Value()
		break
	}
	if sent.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("tx sent without value")
	}
	return sent, nil
}

var ErrReceiptNotFound = errors.New("not found")

// gethCheckTxStatus checks if a signer pending tx is finalized (mined/sealed)
// tx status: 0 means failed, 1 means success
// for go-ethereum nodes
func (f *Faucet) gethCheckTxStatus(ctx context.Context, txHash ethcommon.Hash) (uint64, error) {
	tctx, cancel := context.WithTimeout(ctx, f.ethereum.Timeout)
	defer cancel()
	// the receipt is not available for pending txs, but as pending txs are
	// cached is possible to check at some point in the future if a tx
	// is executed or not
	receipt, err := f.ethereum.Client.TransactionReceipt(tctx, txHash)
	if err != nil {
		// tx receipt not found, probably the tx is in the node mempool
		if err == ErrReceiptNotFound {
			return 0, err
		}
		return 0, fmt.Errorf("cannot check tx status with hash: %s, error: %w", txHash.Hex(), err)
	}
	return receipt.Status, nil
}

var ErrUnprocessedTx = errors.New("unprocessed tx")

// openEthereymCheckTxStatus checks if a signer pending tx is finalized (mined/sealed)
// tx status: 0 means failed, 1 means success
// for go-ethereum nodes
func (f *Faucet) openEthereumCheckTxStatus(ctx context.Context, txHash ethcommon.Hash) (uint64, error) {
	tctx, cancel := context.WithTimeout(ctx, f.ethereum.Timeout)
	defer cancel()
	// the receipt is available for pending txs, status null means pending for execution
	// create request
	// see: https://openethereum.github.io/JSONRPC-eth-module.html#eth_gettransactionreceipt
	resp := object{}
	params := array{}
	params[0] = txHash.Hex()
	// send RPC call
	if err := f.ethereum.RPC.CallContext(tctx,
		&resp,
		"eth_getTransactionReceipt",
		params...,
	); err != nil {
		return 0, fmt.Errorf("RPC call failed: %s", err)
	}
	result, _ := resp["result"].(object)
	status := new(string)
	*status, _ = result["status"].(string)
	if status == nil {
		// tx receipt status null, probably the tx is in the node mempool
		return 0, ErrUnprocessedTx
	}
	if *status == "0x0" {
		return 0, nil
	}
	return 1, nil
}

func (f *Faucet) openEthereumRefreshSignersPendingTx(ctx context.Context) {
	for {
		// for each faucet signer get the pending tx
		for addr, signer := range f.signersWithPendingTxPool {
			// signer does not have pending tx
			if signer.pendingTx == nil {
				continue
			}
			tctx, cancel := context.WithTimeout(ctx, f.ethereum.Timeout)
			defer cancel()
			// get signer pending tx
			// check if tx mined
			// checkTxStatus will return 0 and ErrUnprocessedTx
			// if the tx can be found, but is unprocessed.
			// Otherwise the tx did not
			// arrive to the node or was dropped from it at some point
			status, err := f.openEthereumCheckTxStatus(tctx, signer.pendingTx.Hash)
			if err != nil {
				if err != ErrUnprocessedTx {
					log.Debug("cannot refresh signer %s, receipt for tx: %s not found with error: %s",
						addr.Hex(),
						signer.pendingTx.Hash.Hex(),
						err)
					continue
				} else {
					mempoolPendingTxs := make([]*pendingTx, 0)
					// check if tx is in the node mempool
					mempoolPendingTxs, err = f.openEthereumPendingMempoolTxs(tctx, addr)
					if err != nil {
						log.Warnf("cannot refresh signer %s, cannot get node mempool txs with error: %s",
							addr.Hex(),
							err)
						continue
					}
					for _, mempoolTx := range mempoolPendingTxs {
						if mempoolTx.Hash == signer.pendingTx.Hash {
							// tx in the mempool do not refresh
							log.Debugf("transaction %s found in the mempool, wait until processed", mempoolTx.Hash)
							break
						} else {
							// tx does not exist in the mempool but should exist
							// tx were dropped or did not arrive, send again
							log.Warnf("signer: %s tx: %s was dropped from the mempool or not received",
								addr.Hex(),
								signer.pendingTx.Hash)
							signer.pendingTx = nil
							break
						}
					}
				}
			} else {
				// tx is finalized delete from signer pending txs
				if status == 1 {
					// tx successful
					log.Infof("signer %s tx %s finalized sucessfully, entity %s received %s funds",
						addr.Hex(),
						signer.pendingTx.Hash.Hex(),
						signer.pendingTx.To.Hex(),
						signer.pendingTx.Value.ToInt().String(),
					)
				} else { // tx failed
					log.Warnf("signer %s tx %s failed", addr.Hex(), signer.pendingTx.Hash.Hex())
				}
				signer.pendingTx = nil
			}
		}
		// wait until refresh again
		time.Sleep(time.Second * 10)
	}
}

func (f *Faucet) gethRefreshSignersPendingTx(ctx context.Context) {
	for {
		// for each faucet signer get the pending tx
		for addr, signer := range f.signersWithPendingTxPool {
			// signer does not have pending tx
			if signer.pendingTx == nil {
				continue
			}
			tctx, cancel := context.WithTimeout(ctx, f.ethereum.Timeout)
			defer cancel()
			// get signer pending tx
			// check if tx mined
			// checkTxStatus will return 0 and ErrReceiptNotFound
			// if the tx cannot be found, this probably means that
			// the tx is in the mempool. Otherwise the tx did not
			// arrive to the node or was dropped from it at some point
			status, err := f.gethCheckTxStatus(tctx, signer.pendingTx.Hash)
			if err != nil {
				if err != ErrReceiptNotFound {
					log.Debug("cannot refresh signer %s, receipt for tx: %s not found with error: %s",
						addr.Hex(),
						signer.pendingTx.Hash.Hex(),
						err)
					continue
				} else {
					mempoolPendingTxs := make([]*pendingTx, 0)
					// check if tx is in the node mempool
					mempoolPendingTxs, err = f.gethPendingMempoolTxs(tctx, addr)
					if err != nil {
						log.Warnf("cannot refresh signer %s, cannot get node mempool txs with error: %s",
							addr.Hex(),
							err)
						continue
					}
					for _, mempoolTx := range mempoolPendingTxs {
						if mempoolTx.Hash == signer.pendingTx.Hash {
							// tx in the mempool do not refresh
							log.Debugf("transaction %s found in the mempool, wait until processed", mempoolTx.Hash)
							break
						} else {
							// tx does not exist in the mempool but should exist
							// tx were dropped or did not arrive, send again
							log.Warnf("signer: %s tx: %s was dropped from the mempool or not received",
								addr.Hex(),
								signer.pendingTx.Hash)
							signer.pendingTx = nil
							break
						}
					}
				}
			} else {
				// tx is finalized delete from signer pending txs
				if status == 1 {
					// tx successful
					log.Infof("signer %s tx %s finalized sucessfully, entity %s received %s funds",
						addr.Hex(),
						signer.pendingTx.Hash.Hex(),
						signer.pendingTx.To.Hex(),
						signer.pendingTx.Value.ToInt().String(),
					)
				} else { // tx failed
					log.Warnf("signer %s tx %s failed", addr.Hex(), signer.pendingTx.Hash.Hex())
				}
				signer.pendingTx = nil
			}
		}
		// wait until refresh again
		time.Sleep(time.Second * 10)
	}
}
