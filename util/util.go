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
