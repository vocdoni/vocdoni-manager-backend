package types

const (
	// DEFAULTFAUCETAMOUNT is the default amount of xdai to send to entities
	// 1 XDAI/ETH (as xDAI is the native token for xDAI chain)
	DEFAULTFAUCETAMOUNT = 1000000000000000000
	// DEFAULTFAUCETGASLIMIT is the deafult gas limit for sending an EVM transaction
	DEFAULTFAUCETGASLIMIT = 1000000 // 1M
	// MAXFAUCETAMOUNT is the maximum balance an entity can have for requesting more
	// tokens to the manager faucet
	MAXFAUCETAMOUNT = 2000000000
)
