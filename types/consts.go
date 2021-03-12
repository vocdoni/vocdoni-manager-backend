package types

import "errors"

const (
	// Eth

	Finney = 1000000000000000 // Milliether in wei
	// Default faucet Amounts
	MainnetDefaultFaucetAmount = Finney
	GoerliDefaultFaucetAmount  = Finney * 1500
	XDAIDefaultFaucetAmount    = Finney * 100

	// DefaultGasLimit is the default gas limit for sending an EVM transaction
	DefaultGasLimit = 1000000 // 1M
)

func DefaultFaucetAmount(name string) (int, error) {
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
