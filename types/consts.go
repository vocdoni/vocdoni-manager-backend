package types

import "errors"

const (
	Wei = 1000000000000000000 // 1 Ether in wei
	// Default faucet Amounts
	MainnetDefaultFaucetAmount = 10000000000000000   // 0.01 ETH
	GoerliDefaultFaucetAmount  = 1500000000000000000 // 1.5 ETH
	XDAIDefaultFaucetAmount    = 500000000000000000  // 0.5 ETH (xDAI native token)
	// Default network gas price
	MainnetDefaultGasPrice = 60000000000 // 60 gwei
	GoerliDefaultGasPrice  = 10000000000 // 10 gwei
	XDAIDefaultGasPrice    = 1000000000  // 1 gwei
	// DefaultGasLimit is the default gas limit in wei for sending an EVM transaction
	DefaultGasLimit = 21000000000000
)

// DefaultFaucetAmount returns the default faucet
// amount to send given a valid network name
func DefaultFaucetAmount(name string) (int64, error) {
	switch name {
	case "mainnet":
		return MainnetDefaultFaucetAmount, nil
	case "goerli":
		return GoerliDefaultFaucetAmount, nil
	case "xdai":
		return XDAIDefaultFaucetAmount, nil
	case "xdaistage":
		return XDAIDefaultFaucetAmount, nil
	case "sokol":
		return GoerliDefaultFaucetAmount, nil
	default:
		return 0, errors.New("chain name not found")
	}
}

// DefaultFaucetGasPrices returns the default faucet
// gas price to use for sending value given a valid network name
func DefaultFaucetGasPrice(name string) (int64, error) {
	switch name {
	case "mainnet":
		return MainnetDefaultGasPrice, nil
	case "goerli":
		return GoerliDefaultGasPrice, nil
	case "xdai":
		return XDAIDefaultGasPrice, nil
	case "xdaistage":
		return XDAIDefaultGasPrice, nil
	case "sokol":
		return XDAIDefaultGasPrice, nil
	default:
		return 0, errors.New("chain name not found")
	}
}
