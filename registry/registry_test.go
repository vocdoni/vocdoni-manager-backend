package registry

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/crypto/signature"
	"gitlab.com/vocdoni/vocdoni-manager-backend/test/testcommon"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

var api testcommon.TestAPI

func TestMain(t *testing.M) {
	api = testcommon.TestAPI{}
	route := ""
	api.Start(nil, &route)
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
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/registry", api.Port), t)
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
