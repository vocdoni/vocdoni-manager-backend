package util

import (
	"encoding/hex"
	"fmt"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/util"
)

func PubKeyToEntityID(pubKey string) ([]byte, error) {
	// retrieve entity ID
	var eid []byte
	var err error
	if eid, err = hex.DecodeString(util.TrimHex(pubKey)); err != nil {
		// log.Warn(err)
		// m.Router.SendError(request, "cannot decode public key")
		// return
		return nil, fmt.Errorf("cannot decode public key")
	}
	return ethereum.HashRaw(eid), nil
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
