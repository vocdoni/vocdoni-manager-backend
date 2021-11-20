package util

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	dvoteutil "go.vocdoni.io/dvote/util"
	"go.vocdoni.io/manager/types"
)

func DecodeJsonMessage(object interface{}, key string, ctx *httprouter.HTTPContext) error {
	bytes, err := base64.StdEncoding.DecodeString(ctx.URLParam(key))
	if err != nil {
		return fmt.Errorf("cannot decode json string: (%s): %v", ctx.URLParam(key), err)
	}
	if err = json.Unmarshal(bytes, object); err != nil {
		return fmt.Errorf("cannot unmarshal object %s: %v", key, err)
	}
	return nil
}

func RetrieveSignaturePubKey(ctx *httprouter.HTTPContext) (signaturePubKey []byte, _ error) {
	// check public key length
	signaturePubKey, err := hex.DecodeString(dvoteutil.TrimHex(ctx.URLParam("signaturePublicKey")))
	if err != nil {
		return []byte{}, fmt.Errorf("cannot decode signature pubKey: %v", err)
	}
	if len(signaturePubKey) != ethereum.PubKeyLengthBytes {
		return []byte{}, fmt.Errorf("invalid public key: %x", signaturePubKey)
	}
	return signaturePubKey, err
}

func RetrieveEntityID(ctx *httprouter.HTTPContext) (entityID []byte, signaturePubKey []byte, err error) {
	// check public key length
	if signaturePubKey, err = RetrieveSignaturePubKey(ctx); err != nil {
		return []byte{}, []byte{}, err
	}
	// retrieve entity ID
	entityID, err = PubKeyToEntityID(signaturePubKey)
	if err != nil {
		return []byte{}, signaturePubKey, fmt.Errorf("cannot recover %x entityID from pubKey: (%v)", signaturePubKey, err)
	}
	return entityID, signaturePubKey, nil
}

func SendResponse(response types.MetaResponse, ctx *httprouter.HTTPContext) error {
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}
	if err = ctx.Send(data); err != nil {
		return err
	}
	return nil
}
