package registry

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/vocdoni-manager-backend/test/testcommon"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

var api testcommon.TestAPI

func TestMain(t *testing.M) {
	api = testcommon.TestAPI{}
	api.Start(nil, "")
	reg := NewRegistry(api.EP.Router, api.DB)
	if err := reg.RegisterMethods(""); err != nil {
		panic(err)
	}
	os.Exit(t.Run())
}

func TestRegister(t *testing.T) {
	var req types.MetaRequest
	var s ethereum.SignKeys
	// generate signing keys
	s.Generate()
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Error(err)
	}

	// without token

	// create register request
	req.Method = "register"
	req.EntityID = "12345123451234"
	req.PubKey, _ = s.HexString()
	mInfo := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo}
	// make request
	resp := wsc.Request(req, &s)
	// check register went successful
	if !resp.Ok {
		t.Error(err)
	}

	// with token

	var s2 ethereum.SignKeys
	// generate signing keys
	s2.Generate()
	var req2 types.MetaRequest
	req2.Token = "fa6d35202c264434abe666a4f4cd6c9f"
	req2.Method = "register"
	req2.EntityID = "12345123451234"
	req2.PubKey, _ = s2.HexString()
	mInfo2 := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req2.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo2}
	// make request
	resp2 := wsc.Request(req2, &s2)
	// check register went successful
	if !resp2.Ok {
		t.Error(err)
	}
}

func TestStatus(t *testing.T) {
	var req types.MetaRequest
	var s ethereum.SignKeys

	// generate signing keys
	s.Generate()
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Error(err)
	}

	// Register user and add member
	req.Method = "register"
	req.EntityID = "12345123451234"
	req.PubKey, _ = s.HexString()
	mInfo := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo}
	resp := wsc.Request(req, &s)
	if !resp.Ok {
		t.Error(err)
	}

	// check user is registered calling status
	var req2 types.MetaRequest
	req2.Method = "status"
	req2.EntityID = "12345123451234"
	resp2 := wsc.Request(req2, &s)
	if !resp2.Ok {
		t.Error(err)
	}
	if !resp2.Status.Registered {
		t.Error("Status.Registered expected to be true")
	}
	if resp2.Status.NeedsUpdate {
		t.Error("Status.NeedsUpdate expected to be false")
	}
}
