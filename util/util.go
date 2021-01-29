package util

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/util"
)

func PubKeyToEntityID(pubKey []byte) ([]byte, error) {
	// retrieve entity ID
	// var address ethcommon.Address
	// var addressBytes []byte
	// var err error
	address, err := ethereum.AddrFromPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("cannot decode public key")
	}
	// if addressBytes, err = hex.DecodeString(util.TrimHex(address.String())); err != nil {
	// 	return nil, fmt.Errorf("error extracting address from public key %v", err)
	// }
	return address.Bytes(), nil
	// return ethereum.HashRaw(addressBytes), nil
}

func ValidPubKey(pubKey []byte) bool {
	return len(pubKey) == ethereum.PubKeyLengthBytes
}

func DecodeCensusID(id string, pubKey []byte) ([]byte, error) {
	var censusID string
	split := strings.Split(id, "/")
	// Check for correct format 0xffdf.../0xfdf5f...
	switch {
	case len(split) == 1:
		censusID = split[0]
	case len(split) == 2: // "0x.../0x.... format"
		// Check that the first component is the correct address
		inputAddressBytes, err := hex.DecodeString(util.TrimHex(split[0]))
		if err != nil {
			return nil, fmt.Errorf("error decoding address: %v", err)
		}
		// recover address from signature
		if recoveredAddress, err := ethereum.AddrFromPublicKey(pubKey); err != nil {
			return nil, fmt.Errorf("cannot extract entity address %v", err)
		} else if string(recoveredAddress.Bytes()) != string(inputAddressBytes) {
			return nil, fmt.Errorf("invalid address in census id")
		}
		censusID = split[1]
	default:
		return nil, fmt.Errorf("invalid census id")
	}
	censusIDBytes, err := hex.DecodeString(util.TrimHex(censusID))
	if err != nil {
		return nil, fmt.Errorf("cannot decode censusID: %v", err)
	}

	return censusIDBytes, nil
}

func UniqueUUIDs(list []uuid.UUID) []uuid.UUID {
	found := make(map[uuid.UUID]bool)
	n := 0
	for _, element := range list {
		if !found[element] {
			list[n] = element
			found[element] = true
			n++
		}
	}
	return list[:n]
}

func HexPrefixed(s string) string {
	if !strings.HasPrefix(s, "0x") {
		return fmt.Sprintf("0x%s", s)
	}
	return s
}
