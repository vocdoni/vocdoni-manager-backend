package registry

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/crypto/signature"
	"gitlab.com/vocdoni/vocdoni-manager-backend/test/testcommon"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

func TestMain(t *testing.M) {
	api := testcommon.TestAPI{}
	api.Start("127.0.0.1", "", 8002)
	reg := NewRegistry(api.EP.Router, api.DB)
	if err := reg.RegisterMethods(""); err != nil {
		panic(err)
	}
	time.Sleep(2 * time.Second)
	os.Exit(t.Run())
}

func TestRegister(t *testing.T) {
	var req types.MetaRequest
	var s signature.SignKeys
	s.Generate()
	wsc, err := testcommon.NewAPIConnection("ws://127.0.0.1:8002/registry", t)
	if err != nil {
		t.Error(err)
	}
	req.Method = "register"
	req.EntityID = "12345123451234"

	mInfo := types.MemberInfo{
		Email: "info@vocdoni.io",
	}

	req.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo}

	resp := wsc.Request(req, &s)
	if !resp.Ok {
		t.Error(resp.Message)
	}
}
