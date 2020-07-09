package registry_test

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
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
	registry := registry.NewRegistry(nil, nil)
	if registry == nil {
		t.Fatal("cannot create registry")
	}
}

func TestRegisterMethods(t *testing.T) {
	// create signer
	signer := new(ethereum.SignKeys)
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
	registry := registry.NewRegistry(r, db)
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
	s := ethereum.SignKeys{}
	s.Generate()
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
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
		t.Fatal(err)
	}

	// invalid pubkey should fail
	req.Method = "register"
	req.EntityID = "12345123451234"
	req.PubKey = "0x0"
	mInfo = types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo}
	// make request
	resp = wsc.Request(req, &s)
	// check register went successful
	if resp.Ok {
		t.Fatal("invalid pubkey should fail")
	}

	// valid pubkey lenght but invalid content and cannot decode
	req.PubKey, _ = s2.HexString()
	req.PubKey = req.PubKey[:len(req.PubKey)-1] + "Z"
	// make request
	resp = wsc.Request(req, &s)
	// check register went successful
	if resp.Ok {
		t.Fatal("invalid pubkey content should fail")
	}

	// should fail if pubkey != extacted signature pubkey
	req.Method = "register"
	req.EntityID = "12345123451234"
	req.PubKey, _ = s2.HexString()
	mInfo = types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo}
	// make request
	resp = wsc.Request(req, &s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if pubkey != extacted signature pubkey")
	}

	// should fail if no pubkey present and signature pubkey cannot be decoded
	req.Method = "register"
	req.EntityID = "12345123451234"
	req.PubKey = ""
	mInfo = types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo}
	// make request
	resp = wsc.Request(req, &s)
	// check register went successful
	if !resp.Ok {
		t.Fatal("should fail if no pubkey present and signature pubkey cannot be decoded")
	}

	// should fail if invalid entityID
	req.Method = "register"
	req.EntityID = "0xZ"
	req.PubKey, _ = s.HexString()
	mInfo = types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo}
	// make request
	resp = wsc.Request(req, &s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if invalid entityID")
	}

	// should fail if no token provided and Member info invalid
	var req3 types.MetaRequest
	req3.Method = "register"
	req3.EntityID = "12345123451234"
	req3.Token = ""
	req3.PubKey, _ = s.HexString()
	mInfo = types.MemberInfo{}
	req3.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo}
	// make request
	resp = wsc.Request(req3, &s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if no token provided and Member info invalid")
	}

	// should fail if add member fails
	var req4 types.MetaRequest
	req4.Method = "register"
	req4.EntityID = "12345123451234"
	req4.PubKey, _ = s.HexString()
	mInfo3 := types.MemberInfo{
		Email: "fail@fail.fail",
	}
	req4.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo3}
	// make request
	resp = wsc.Request(req4, &s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if add member fails")
	}

	// should fail if token is an invalid hex string
	var req5 types.MetaRequest
	req5.Method = "register"
	req5.EntityID = "12345123451234"
	req5.Token = "0xABCZ"
	req5.PubKey, _ = s.HexString()
	mInfo5 := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req5.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo5}
	// make request
	resp = wsc.Request(req5, &s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if token is an invalid hex string")
	}

	// should fail if token cannot be decoded even if is a valid hex string
	var req6 types.MetaRequest
	req6.Method = "register"
	req6.EntityID = "12345123451234"
	req6.Token = "0x"
	req6.PubKey, _ = s.HexString()
	mInfo6 := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req6.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo6}
	// make request
	resp = wsc.Request(req6, &s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if token cannot be decoded even if is a valid hex string")
	}

	// should fail if entity does not exist
	var req7 types.MetaRequest
	req7.Method = "register"
	req7.EntityID = "f6da3e4864d566faf82163a407e84a9001592678"
	req7.PubKey, _ = s.HexString()
	mInfo7 := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req7.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo7}
	// make request
	resp = wsc.Request(req7, &s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if entity does not exist")
	}

	// should fail if req.entityID != fetched entity.ID
	var req8 types.MetaRequest
	req8.Method = "register"
	req8.EntityID = "ca526af2aaa0f3e9bb68ab80de4392590f7b153a"
	req8.PubKey, _ = s.HexString()
	mInfo8 := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req8.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo8}
	// make request
	resp = wsc.Request(req8, &s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if req.entityID != fetched entity.ID")
	}

	// if user does not exist create
	var req9 types.MetaRequest
	constSigner := new(ethereum.SignKeys)
	constSigner.AddHexKey(testdb.Signers[0].Priv)
	req9.Method = "register"
	req9.EntityID = "12345123451234"
	req9.PubKey, _ = constSigner.HexString()
	mInfo9 := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req9.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo9}
	// make request
	resp = wsc.Request(req9, constSigner)
	// check register went successful
	if !resp.Ok {
		t.Fatal("should create user")
	}

	// should fail if user does not exist and fails on create
	var req10 types.MetaRequest
	constSigner2 := new(ethereum.SignKeys)
	constSigner2.AddHexKey(testdb.Signers[1].Priv)
	//p1, p2 := constSigner2.HexString()
	//t.Fatalf("%s : %s", p1, p2)
	req10.Method = "register"
	req10.EntityID = "12345123451234"
	req10.PubKey, _ = constSigner2.HexString()
	mInfo10 := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req10.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo10}
	// make request
	resp = wsc.Request(req10, constSigner2)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail on addUser")
	}

	// should fail cannot query for user
	var req11 types.MetaRequest
	constSigner3 := new(ethereum.SignKeys)
	constSigner3.AddHexKey(testdb.Signers[2].Priv)
	//p1, p2 := constSigner3.HexString()
	//t.Fatalf("%s : %s", p1, p2)
	req11.Method = "register"
	req11.EntityID = "12345123451234"
	req11.PubKey, _ = constSigner3.HexString()
	mInfo11 := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req11.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo11}
	// make request
	resp = wsc.Request(req11, constSigner3)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail on query user")
	}
}

func TestStatus(t *testing.T) {
	var req types.MetaRequest
	var s ethereum.SignKeys

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
	mInfo := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo}
	resp := wsc.Request(req, &s)
	if !resp.Ok {
		t.Fatal(err)
	}

	// check user is registered calling status
	var req2 types.MetaRequest
	req2.Method = "status"
	req2.EntityID = "12345123451234"
	resp2 := wsc.Request(req2, &s)
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
	req3.Method = "status"
	req3.EntityID = "0xZ"
	resp3 := wsc.Request(req3, &s)
	if resp3.Ok {
		t.Fatal(err)
	}

	// should fail if entity does not exist
	var req4 types.MetaRequest
	req4.Method = "status"
	req4.EntityID = "f6da3e4864d566faf82163a407e84a9001592678"
	resp4 := wsc.Request(req4, &s)
	if resp4.Ok {
		t.Fatal("should fail if entity not found")
	}

	// registered should be false if user not found
	var req5 types.MetaRequest
	constSigner := new(ethereum.SignKeys)
	constSigner.AddHexKey(testdb.Signers[0].Priv)
	req5.Method = "status"
	req5.EntityID = "12345123451234"
	resp5 := wsc.Request(req5, constSigner)
	if resp5.Ok {
		t.Fatal("should fail if user not found")
	}

	// registered should be false if user is not a member
	var req6 types.MetaRequest
	constSigner2 := new(ethereum.SignKeys)
	constSigner2.AddHexKey(testdb.Signers[3].Priv)
	req6.Method = "status"
	req6.EntityID = "12345123451234"
	resp6 := wsc.Request(req6, constSigner2)
	if resp6.Ok {
		if resp6.Status.Registered != false {
			t.Fatal("registered should be false if user is not a member")
		}
	}
}

func TestSubscribe(t *testing.T) {
	var req types.MetaRequest
	var s ethereum.SignKeys
	// generate signing keys
	s.Generate()
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}
	req.Method = "subscribe"
	resp := wsc.Request(req, &s)
	if !resp.Ok {
		t.Fatal(err)
	}
}
func TestUnsubscribe(t *testing.T) {
	var req types.MetaRequest
	var s ethereum.SignKeys
	// generate signing keys
	s.Generate()
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}
	req.Method = "unsubscribe"
	resp := wsc.Request(req, &s)
	if !resp.Ok {
		t.Fatal(err)
	}
}
