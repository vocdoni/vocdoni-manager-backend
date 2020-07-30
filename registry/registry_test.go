package registry_test

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/net"
	gtypes "gitlab.com/vocdoni/go-dvote/types"
	"gitlab.com/vocdoni/manager/manager-backend/database/testdb"
	"gitlab.com/vocdoni/manager/manager-backend/registry"
	"gitlab.com/vocdoni/manager/manager-backend/router"
	"gitlab.com/vocdoni/manager/manager-backend/test/testcommon"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

var api testcommon.TestAPI

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	api = testcommon.TestAPI{Port: 12000 + rand.Intn(1000)}
	api.Start(nil, "/api")
	os.Exit(m.Run())
}

func TestNewRegistry(t *testing.T) {
	registry := registry.NewRegistry(nil, nil, nil)
	if registry == nil {
		t.Fatal("cannot create registry")
	}
}

func TestRegisterMethods(t *testing.T) {
	// create signer
	signer := ethereum.NewSignKeys()
	if err := signer.Generate(); err != nil {
		t.Fatalf("cannot generate signer: %v", err)
	}
	// create proxy
	pxy := net.NewProxy()
	pxy.C.Address = "127.0.0.1"
	pxy.C.Port = 0
	// init proxy
	if err := pxy.Init(); err != nil {
		t.Fatalf("cannot init proxy: %v", err)
	}
	// create router channel
	listenerOutput := make(chan gtypes.Message)
	// create ws
	ws := new(net.WebsocketHandle)
	ws.Init(new(gtypes.Connection))
	ws.SetProxy(pxy)
	// init router
	r := router.InitRouter(listenerOutput, ws, signer)
	// create database
	db, err := testdb.New()
	if err != nil {
		t.Fatalf("cannot create DB: %v", err)
	}
	// create registry
	registry := registry.NewRegistry(r, db, nil)
	// register methods
	if err := registry.RegisterMethods(""); err != nil {
		t.Fatalf("cannot register methods: %v", err)

	}
}

func TestSend(t *testing.T) {
	// nothing to test here, router layer
}

func TestRegister(t *testing.T) {
	var req types.MetaRequest
	// generate signing keys
	s := ethereum.NewSignKeys()
	s.Generate()
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// create register request
	req.Method = "register"
	req.EntityID = "12345123451234"
	req.MemberInfo = &types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if !resp.Ok {
		t.Fatal(err)
	}

	s2 := ethereum.NewSignKeys()
	// generate signing keys
	s2.Generate()
	var req2 types.MetaRequest
	req2.Method = "register"
	req2.EntityID = "12345123451234"
	req2.MemberInfo = &types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if !resp2.Ok {
		t.Fatal(err)
	}

	// should fail if invalid entityID
	req.Method = "register"
	req.EntityID = "0xZ"
	req.MemberInfo = &types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	// make request
	resp = wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if invalid entityID")
	}

	// should fail if add member fails
	var req4 types.MetaRequest
	req4.Method = "register"
	req4.EntityID = "12345123451234"
	req4.MemberInfo = &types.MemberInfo{
		Email: "fail@fail.fail",
	}
	// make request
	resp = wsc.Request(req4, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if add member fails")
	}

	// should fail if entity does not exist
	var req7 types.MetaRequest
	req7.Method = "register"
	req7.EntityID = "f6da3e4864d566faf82163a407e84a9001592678"
	req7.MemberInfo = &types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	// make request
	resp = wsc.Request(req7, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if entity does not exist")
	}

	// TODO: Enable if separate select query for Entity
	// should fail if req.entityID != fetched entity.ID
	// var req8 types.MetaRequest
	// req8.Method = "register"
	// req8.EntityID = "ca526af2aaa0f3e9bb68ab80de4392590f7b153a"
	// req8.MemberInfo = &types.MemberInfo{
	// 	Email: "info@vocdoni.io",
	// }
	// resp = wsc.Request(req8, s)
	// // check register went successful
	// if resp.Ok {
	// 	t.Fatal("should fail if req.entityID != fetched entity.ID")
	// }

	// if user does not exist create
	var req9 types.MetaRequest
	constSigner := ethereum.NewSignKeys()
	constSigner.AddHexKey(testdb.Signers[0].Priv)
	req9.Method = "register"
	req9.EntityID = "12345123451234"
	req9.MemberInfo = &types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	// make request
	resp = wsc.Request(req9, constSigner)
	// check register went successful
	if !resp.Ok {
		t.Fatal("should create user")
	}

	// should fail if user does not exist and fails on create
	// TODO: Update for uncompressed pubkey
	// var req10 types.MetaRequest
	// constSigner2 := ethereum.NewSignKeys()
	// constSigner2.AddHexKey(testdb.Signers[1].Priv)
	// req10.Method = "register"
	// req10.EntityID = "12345123451234"
	// req10.MemberInfo = &types.MemberInfo{
	// 	Email: "info@vocdoni.io",
	// }
	// // make request
	// resp = wsc.Request(req10, constSigner2)
	// // check register went successful
	// if resp.Ok {
	// 	t.Fatal("should fail on addUser")
	// }

	// should fail cannot query for user
	// TODO: Update for uncompressed pubkey
	// var req11 types.MetaRequest
	// constSigner3 := ethereum.NewSignKeys()
	// constSigner3.AddHexKey(testdb.Signers[2].Priv)
	// //p1, p2 := constSigner3.HexString()
	// //t.Fatalf("%s : %s", p1, p2)
	// req11.Method = "register"
	// req11.EntityID = "12345123451234"
	// req11.MemberInfo = &types.MemberInfo{
	// 	Email: "info@vocdoni.io",
	// }
	// // make request
	// resp = wsc.Request(req11, constSigner3)
	// // check register went successful
	// if resp.Ok {
	// 	t.Fatal("should fail on query user")
	// }
}

func TestStatus(t *testing.T) {
	var req types.MetaRequest
	s := ethereum.NewSignKeys()

	// generate signing keys
	s.Generate()
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// Register user and add member
	req.Method = "register"
	req.EntityID = "12345123451234"
	req.PubKey, _ = s.HexString()
	req.MemberInfo = &types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	resp := wsc.Request(req, s)
	if !resp.Ok {
		t.Fatal(err)
	}

	// check user is registered calling status
	var req2 types.MetaRequest
	req2.Method = "registrationStatus"
	req2.EntityID = "12345123451234"
	resp2 := wsc.Request(req2, s)
	if !resp2.Ok {
		t.Fatal(err)
	}
	if !resp2.Status.Registered {
		t.Fatal("Status.Registered expected to be true")
	}
	if resp2.Status.NeedsUpdate {
		t.Fatal("Status.NeedsUpdate expected to be false")
	}

	// should fail if invalid entityID
	var req3 types.MetaRequest
	req3.Method = "registrationStatus"
	req3.EntityID = "0xZ"
	resp3 := wsc.Request(req3, s)
	if resp3.Ok {
		t.Fatal(err)
	}

	// should fail if entity does not exist
	var req4 types.MetaRequest
	req4.Method = "registrationStatus"
	req4.EntityID = "f6da3e4864d566faf82163a407e84a9001592678"
	resp4 := wsc.Request(req4, s)
	if resp4.Ok {
		t.Fatal("should fail if entity not found")
	}

	// registered should be false if user is not a member
	// TODO: Update for uncompressed pubkey
	// var req6 types.MetaRequest
	// constSigner2 := ethereum.NewSignKeys()
	// constSigner2.AddHexKey(testdb.Signers[3].Priv)
	// req6.Method = "registrationStatus"
	// req6.EntityID = "12345123451234"
	// resp6 := wsc.Request(req6, constSigner2)
	// if resp6.Ok {
	// 	if resp6.Status.Registered != false {
	// 		t.Fatal("registered should be false if user is not a member")
	// 	}
	// }
}

func TestSubscribe(t *testing.T) {
	var req types.MetaRequest
	s := ethereum.NewSignKeys()
	// generate signing keys
	s.Generate()
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}
	req.Method = "subscribe"
	resp := wsc.Request(req, s)
	if !resp.Ok {
		t.Fatal(err)
	}
}
func TestUnsubscribe(t *testing.T) {
	var req types.MetaRequest
	s := ethereum.NewSignKeys()
	// generate signing keys
	s.Generate()
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}
	req.Method = "unsubscribe"
	resp := wsc.Request(req, s)
	if !resp.Ok {
		t.Fatal(err)
	}
}
