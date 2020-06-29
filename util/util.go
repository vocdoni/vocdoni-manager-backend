package util

import (
	"encoding/hex"
	"fmt"
	"strings"

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

func DecodeCensusID(id string, pubKey string) ([]byte, error) {
	var err error
	split := strings.Split(id, "/")
	iter := 0
	// Check for correct format 0xffdf.../0xfdf5f...
	if len(split) > 2 || len(split) < 1 {
		return nil, fmt.Errorf("invalid census ID format")
	}
	if len(split) == 2 { // "0x.../0x.... format"
		addressIn := util.TrimHex(split[iter])
		iter += iter

		// Check that the first component is the correct address
		if !util.IsHex(addressIn) {
			return nil, fmt.Errorf("invalid census ID format")
		}

		var address []byte
		if address, err = PubKeyToAddress(pubKey); err != nil {
			return nil, fmt.Errorf("cannot extract entity address %+v", err)
		}
		if hex.EncodeToString(address) != addressIn {
			return nil, fmt.Errorf("invalid census id")
		}
	}
	censusIDIn := util.TrimHex(split[iter])
	if !util.IsHex(censusIDIn) {
		return nil, fmt.Errorf("invalid census ID format")
	}

	var censusID []byte
	if censusID, err = hex.DecodeString(censusIDIn); err != nil {
		return nil, fmt.Errorf("cannot decode censusID: %+v", err)
	}
	return censusID, nil
}
