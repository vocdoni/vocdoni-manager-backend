package registry

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/crypto/signature"
	"gitlab.com/vocdoni/vocdoni-manager-backend/test"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

func TestRegister(t *testing.T) {
	api := test.TestAPI{}
	api.Start(t, "127.0.0.1", "", 8002)

	reg := NewRegistry(api.EP.Router, api.DB)
	if err := reg.RegisterMethods(""); err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)
	wsc := test.NewAPIConnection(t, "ws://127.0.0.1:8002/registry")
	var req types.MetaRequest
	var s signature.SignKeys
	s.Generate()
	req.Method = "register"
	req.EntityID = "12345123451234"
	req.Member = &types.Member{ID: uuid.New()}
	resp := wsc.Request(req, &s)
	if !resp.Ok {
		t.Error(resp.Message)
	}
}
