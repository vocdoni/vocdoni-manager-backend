package manager

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
	if err := api.Start("127.0.0.1", "", 8002); err != nil {
		panic(err)
	}
	reg := NewManager(api.EP.Router, api.DB)
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
	wsc, err := testcommon.NewAPIConnection("ws://127.0.0.1:8002/manager", t)
	if err != nil {
		t.Error(err)
	}

	req.Method = "register"
	req.EntityID = "12345123451234"
	req.Member = &types.Member{ID: uuid.New()}
	resp := wsc.Request(req, &s)
	if !resp.Ok {
		t.Error(resp.Message)
	}
}
