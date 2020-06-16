package util

import (
	"encoding/hex"
	"fmt"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/util"
)

func PubKeyToEntityID(pubKey string) ([]byte, error) {
	// retrieve entity ID
	var address []byte
	var err error
	if address, err = PubKeyToAddress(pubKey); err != nil {
		return nil, fmt.Errorf("cannot decode public key")
	}
	return ethereum.HashRaw(address), nil
}

func PubKeyToAddress(pubKey string) ([]byte, error) {
	var address string
	var err error
	var addressB []byte
	if address, err = ethereum.AddrFromPublicKey(pubKey); err != nil {
		return nil, fmt.Errorf("Error extracting address from public key %w", err)
	}
	if addressB, err = hex.DecodeString(util.TrimHex(address)); err != nil {
		return nil, fmt.Errorf("Error extracting address from public key %w", err)
	}
	return addressB, nil
}

func SignerEntityAddress(singer ethereum.SignKeys) ([]byte, error) {
	// retrieve entity ID
	var address []byte
	var err error
	addressString := singer.EthAddrString()
	if address, err = hex.DecodeString(addressString); err != nil {
		return nil, fmt.Errorf("cannot decode entityAddress")
	}
	return address, nil
}

func SignerEntityID(singer ethereum.SignKeys) ([]byte, error) {
	// retrieve entity ID
	var address []byte
	var err error
	if address, err = SignerEntityAddress(singer); err != nil {
		return nil, fmt.Errorf("cannot decode entityAddress")
	}
	return ethereum.HashRaw(address), nil
}
