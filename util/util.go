package util

import (
	"encoding/hex"
	"fmt"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/util"
)

func PubKeyToEntityID(pubKey string) ([]byte, error) {
	// retrieve entity ID
	var address ethcommon.Address
	var addressBytes []byte
	var err error
	if address, err = PubKeyToAddress(pubKey); err != nil {
		return nil, fmt.Errorf("cannot decode public key")
	}
	if addressBytes, err = hex.DecodeString(util.TrimHex(address.String())); err != nil {
		return nil, fmt.Errorf("error extracting address from public key %v", err)
	}
	return ethereum.HashRaw(addressBytes), nil
}

func PubKeyToAddress(pubKey string) (ethcommon.Address, error) {
	var address ethcommon.Address
	var err error
	if address, err = ethereum.AddrFromPublicKey(pubKey); err != nil {
		return ethcommon.Address{}, fmt.Errorf("error extracting address from public key %v", err)
	}
	return address, nil
}

func DecodeCensusID(id string, pubKey string) ([]byte, error) {
	var censusID string
	split := strings.Split(id, "/")
	// Check for correct format 0xffdf.../0xfdf5f...
	switch {
	case len(split) == 1:
		censusID = split[0]
	case len(split) == 2: // "0x.../0x.... format"
		// Check that the first component is the correct address
		if !util.IsHex(util.TrimHex(split[0])) {
			return nil, fmt.Errorf("invalid census ID format")
		}
		if address, err := ethereum.AddrFromPublicKey(pubKey); err != nil {
			return nil, fmt.Errorf("cannot extract entity address %+v", err)
		} else if address.String() != split[0] {
			return nil, fmt.Errorf("invalid census id")
		}
		censusID = split[1]
	default:
		return nil, fmt.Errorf("invalid census id")
	}
	censusIDIn := util.TrimHex(censusID)
	if !util.IsHex(censusIDIn) {
		return nil, fmt.Errorf("invalid census ID format")
	}
	censusIDBytes, err := hex.DecodeString(censusIDIn)
	if err != nil {
		return nil, fmt.Errorf("cannot decode censusID: %+v", err)
	}

	return censusIDBytes, nil
}
