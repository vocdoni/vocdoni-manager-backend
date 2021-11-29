package manager_test

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/google/uuid"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/multirpc/transports"
	"go.vocdoni.io/dvote/multirpc/transports/mhttp"
	"go.vocdoni.io/manager/database/testdb"
	"go.vocdoni.io/manager/manager"
	"go.vocdoni.io/manager/router"
	"go.vocdoni.io/manager/test/testcommon"
	"go.vocdoni.io/manager/types"
)

var api testcommon.TestAPI

func TestMain(m *testing.M) {
	api = testcommon.TestAPI{Port: 12000 + rand.Intn(1000)}
	api.Start(nil, "/api")
	os.Exit(m.Run())
}

func TestNewManager(t *testing.T) {
	if mgr := manager.NewManager(nil, nil, nil); mgr == nil {
		t.Fatal("cannot create manager")
	}
}

func TestRegisterMethods(t *testing.T) {
	// create signer
	signer := ethereum.NewSignKeys()
	if err := signer.Generate(); err != nil {
		t.Fatalf("cannot generate signer: %v", err)
	}
	// create proxy
	pxy := mhttp.NewProxy()
	pxy.Conn.Address = "127.0.0.1"
	pxy.Conn.Port = 0
	// init proxy
	if err := pxy.Init(); err != nil {
		t.Fatalf("cannot init proxy: %v", err)
	}
	// create router channel
	listenerOutput := make(chan transports.Message)
	// create ws
	ws := new(mhttp.WebsocketHandle)
	ws.Init(new(transports.Connection))
	ws.SetProxy(pxy)
	// create transports map
	ts := make(map[string]transports.Transport)
	ts["ws"] = ws
	// init router
	r := router.InitRouter(listenerOutput, ts, signer)
	// create database
	db, err := testdb.New()
	if err != nil {
		t.Fatalf("cannot create DB: %v", err)
	}
	// create manager
	manager := manager.NewManager(db, nil, nil)
	// register methods
	if err := manager.RegisterMethods(""); err != nil {
		t.Fatalf("cannot register methods: %v", err)
	}
}

func TestSend(t *testing.T) {
	// nothing to test here, router layer
}

func TestSignUp(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if addEntity fails
	var req types.MetaRequest
	// generate signing keys
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	//eid, err := util.PubKeyToEntityID(testdb.Signers[0].Pub)
	//t.Fatalf("%s", hex.EncodeToString(eid))
	req.Method = "signUp"
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if cannot addEntity")
	}

	// should fail if AddTarget fails
	var req2 types.MetaRequest
	// generate signing keys
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[1].Priv)
	//eid, err := util.PubKeyToEntityID(testdb.Signers[1].Pub)
	//t.Fatalf("%s", hex.EncodeToString(eid))
	req2.Method = "signUp"
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if resp2.Ok {
		t.Fatal("should fail if cannot addEntity")
	}

	// should success if entity and target can be added
	// should fail if AddTarget fails
	var req3 types.MetaRequest
	// generate signing keys
	s3 := ethereum.NewSignKeys()
	s3.AddHexKey(testdb.Signers[2].Priv)
	//eid, err := util.PubKeyToEntityID(testdb.Signers[1].Pub)
	//t.Fatalf("%s", hex.EncodeToString(eid))
	req3.Method = "signUp"
	// make request
	resp3 := wsc.Request(req3, s3)
	// check register went successful
	if !resp3.Ok {
		t.Fatal("should signUp successful")
	}
}

func TestGetEntity(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if Entity fails
	var req types.MetaRequest
	// generate signing keys
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	//eid, err := util.PubKeyToEntityID(testdb.Signers[0].Pub)
	//t.Fatalf("%s", hex.EncodeToString(eid))
	req.Method = "getEntity"
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if entity does not exist")
	}

	// should success if entity and target can be added
	// should fail if AddTarget fails
	var req3 types.MetaRequest
	// generate signing keys
	s3 := ethereum.NewSignKeys()
	s3.AddHexKey(testdb.Signers[2].Priv)
	//eid, err := util.PubKeyToEntityID(testdb.Signers[1].Pub)
	//t.Fatalf("%s", hex.EncodeToString(eid))
	req3.Method = "getEntity"
	// make request
	resp3 := wsc.Request(req3, s3)
	// check register went successful
	if !resp3.Ok {
		t.Fatal("should signUp successful")
	}
}

func TestListMembers(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if checkOptions returns an error when order invalid
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "0x",
		Skip:   2,
		SortBy: "lastName",
	}
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if invalid ListOptions order")
	}

	// should fail db list members does not return any row
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[1].Priv)
	var req2 types.MetaRequest
	req2.Method = "listMembers"
	req2.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "ascend",
		Skip:   2,
		SortBy: "lastName",
	}
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if resp2.Ok {
		t.Fatal("should fail if not members found")
	}

	// should fail db list members fails
	var req3 types.MetaRequest
	req3.Method = "listMembers"
	req3.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "ascend",
		Skip:   2,
		SortBy: "lastName",
	}
	// make request
	resp3 := wsc.Request(req3, s)
	// check register went successful
	if resp3.Ok {
		t.Fatal("should fail if db listMembers fail")
	}

	// should success if all correct
	s3 := ethereum.NewSignKeys()
	s3.AddHexKey(testdb.Signers[3].Priv)
	var req4 types.MetaRequest
	req4.Method = "listMembers"
	req4.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "ascend",
		Skip:   2,
		SortBy: "lastName",
	}
	// make request
	resp4 := wsc.Request(req4, s3)
	// check register went successful
	if !resp4.Ok {
		t.Fatal("should success if all correct")
	}
}

func TestGetMember(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if no member found
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[1].Priv)
	var req types.MetaRequest
	req.Method = "getMember"
	u := uuid.New()
	req.MemberID = &u
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if db Member() returns no rows")
	}

	// should fail if cannot get member from db
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[0].Priv)
	var req2 types.MetaRequest
	req2.Method = "getMember"
	u = uuid.New()
	req2.MemberID = &u
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if resp2.Ok {
		t.Fatal("should fail if cannot get member from db")
	}

	// should fail if listTargets returns no rows
	s3 := ethereum.NewSignKeys()
	s3.AddHexKey(testdb.Signers[2].Priv)
	//eid, err := util.PubKeyToEntityID(testdb.Signers[2].Pub)
	//t.Fatalf("%s", hex.EncodeToString(eid))
	var req3 types.MetaRequest
	req3.Method = "getMember"
	u = uuid.New()
	req3.MemberID = &u
	// make request
	resp3 := wsc.Request(req3, s2)
	// check register went successful
	if resp3.Ok {
		t.Fatal("should fail listTargets returns no rows")
	}

	// should fail if listTargets fail
	s4 := ethereum.NewSignKeys()
	s4.AddHexKey(testdb.Signers[3].Priv)
	var req4 types.MetaRequest
	req4.Method = "getMember"
	u = uuid.New()
	req4.MemberID = &u
	// make request
	resp4 := wsc.Request(req4, s4)
	// check register went successful
	if resp4.Ok {
		t.Fatal("should fail if db list target fails")
	}

	// should success if previous succesful
	s5 := ethereum.NewSignKeys()
	s5.Generate()
	var req5 types.MetaRequest
	req5.Method = "getMember"
	u = uuid.New()
	req5.MemberID = &u
	// make request
	resp5 := wsc.Request(req5, s5)
	// check register went successful
	if !resp5.Ok {
		t.Fatalf("should list succesful: %s", resp5.Message)
	}
}

func TestUpdateMember(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if no member found
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "updateMember"
	req.Member = &types.Member{}
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if update fails on db")
	}

	// should update member
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[1].Priv)
	var req2 types.MetaRequest
	req2.Method = "updateMember"
	req2.Member = &types.Member{}
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if !resp2.Ok {
		t.Fatal("should update member")
	}
}

func TestDeleteMembers(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if db delete member fails
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "deleteMembers"
	req.MemberIDs = []uuid.UUID{uuid.New()}
	// *req.MemberID = uuid.New()
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if update fails on db")
	}

	// otherwise should success
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[1].Priv)
	var req2 types.MetaRequest
	req2.Method = "deleteMembers"
	req2.MemberIDs = []uuid.UUID{uuid.New()}
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if !resp2.Ok {
		t.Fatal("should succeed")
	}
}

func TestCountMembers(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if db count members fails
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "countMembers"
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if update fails on db")
	}

	// otherwise should success
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[1].Priv)
	var req2 types.MetaRequest
	req2.Method = "countMembers"
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if !resp2.Ok {
		t.Fatal("should success")
	}
}

func TestGenerateTokens(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if amount is less or equal to 0
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "generateTokens"
	req.Amount = 0
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if invalid amount")
	}

	// should fail if cannot generate members with tokens
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[1].Priv)
	var req2 types.MetaRequest
	req2.Method = "generateTokens"
	req2.Amount = 2
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if resp2.Ok {
		t.Fatal("should fail if cannot generate members")
	}

	// otherwise should success
	s3 := ethereum.NewSignKeys()
	s3.AddHexKey(testdb.Signers[2].Priv)
	var req3 types.MetaRequest
	req3.Method = "generateTokens"
	req3.Amount = 2
	// make request
	resp3 := wsc.Request(req3, s3)
	// check register went successful
	if !resp3.Ok {
		t.Fatal("should success")
	}
}

func TestExportTokens(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if db MembersTokensEmails fails
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "exportTokens"
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if db members tokens emails fails")
	}

	// should fail if no rows returned from db MembersTokensEmails
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[1].Priv)
	var req2 types.MetaRequest
	req2.Method = "exportTokens"
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if resp2.Ok {
		t.Fatal("should fail if db members tokens emails returns no rows")
	}

	// otherwise should sucesss
	// should fail if db MembersTokensEmails fails
	s3 := ethereum.NewSignKeys()
	s3.AddHexKey(testdb.Signers[2].Priv)
	var req3 types.MetaRequest
	req3.Method = "exportTokens"
	// make request
	resp3 := wsc.Request(req3, s3)
	// check register went successful
	if !resp3.Ok {
		t.Fatal("should success")
	}
}

func TestImportMembers(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if members info < 1
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "importMembers"
	req.MembersInfo = []types.MemberInfo{}
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if members info < 1")
	}

	// should fail if db import members fails
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[1].Priv)
	var req2 types.MetaRequest
	req2.Method = "importMembers"
	req2.MembersInfo = make([]types.MemberInfo, 2)
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if resp2.Ok {
		t.Fatal("should fail if db import members fails")
	}

	// otherwise should success
	var req3 types.MetaRequest
	req3.Method = "importMembers"
	req3.MembersInfo = make([]types.MemberInfo, 2)
	// make request
	resp3 := wsc.Request(req3, s)
	// check register went successful
	if !resp3.Ok {
		t.Fatal("should success")
	}
}

func TestCountTargets(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if db countTargets fails
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "countTargets"
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if db count targets fails")
	}

	// otherwise should success
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[1].Priv)
	var req2 types.MetaRequest
	req2.Method = "countTargets"
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if !resp2.Ok {
		t.Fatal("should success")
	}
}

func TestListTargets(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if invalid list options
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "listTargets"
	req.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "0x",
		Skip:   2,
		SortBy: "lastName",
	}
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if invalid list options")
	}

	// should fail if listTargets returns no rows
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[1].Priv)
	var req2 types.MetaRequest
	req2.Method = "listTargets"
	req2.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "ascend",
		Skip:   2,
		SortBy: "lastName",
	}
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if resp2.Ok {
		t.Fatal("should fail if not members found")
	}

	// should fail if db listTargets fails
	// should fail db list members fails
	var req3 types.MetaRequest
	req3.Method = "listTargets"
	req3.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "ascend",
		Skip:   2,
		SortBy: "lastName",
	}
	// make request
	resp3 := wsc.Request(req3, s)
	// check register went successful
	if resp3.Ok {
		t.Fatal("should fail if db list targets fail")
	}

}

func TestGetTarget(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if target not found
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "getTarget"
	req.TargetID = new(uuid.UUID)
	*req.TargetID = uuid.New()
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if db Target fails")
	}

	// should fail if db targetMembers fail
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[1].Priv)
	var req2 types.MetaRequest
	req2.Method = "getTarget"
	req2.TargetID = new(uuid.UUID)
	*req2.TargetID = uuid.New()
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if resp2.Ok {
		t.Fatal("should fail if db Target members fails")
	}

	// otherwise should success
	s3 := ethereum.NewSignKeys()
	s3.AddHexKey(testdb.Signers[2].Priv)
	var req3 types.MetaRequest
	req3.Method = "getTarget"
	req3.TargetID = new(uuid.UUID)
	*req3.TargetID = uuid.New()
	// make request
	resp3 := wsc.Request(req3, s3)
	// check register went successful
	if !resp3.Ok {
		t.Fatal("should success")
	}
}

func TestDumpTarget(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if db target fails
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "dumpTarget"
	req.TargetID = new(uuid.UUID)
	*req.TargetID = uuid.New()
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if db Target fails")
	}

	// should fail if db dumpClaims fails
	/*
		s2 := ethereum.NewSignKeys()
		s2.AddHexKey(testdb.Signers[1].Priv)
		var req2 types.MetaRequest
		req2.Method = "dumpTarget"
		req2.TargetID = uuid.New()
		// make request
		resp2 := wsc.Request(req2, s2)
		// check register went successful
		if resp2.Ok {
			t.Fatal("should fail if db Target fails")
		}
	*/

	// otherwise should success
	/*
		s3 := ethereum.NewSignKeys()
		s3.AddHexKey(testdb.Signers[2].Priv)
		var req3 types.MetaRequest
		req3.Method = "dumpTarget"
		req3.TargetID = uuid.New()
		// make request
		resp3 := wsc.Request(req3, s3)
		// check register went successful
		if !resp3.Ok {
			t.Fatal("should success")
		}
	*/
}

func TestAddCensus(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	// should fail if len(targetID) == 0
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "addCensus"
	req.TargetID = new(uuid.UUID)
	*req.TargetID = uuid.Nil
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if len(targetId) == 0")
	}

	// should fail if len(censusID) == 0
	var req2 types.MetaRequest
	req2.Method = "addCensus"
	req2.TargetID = new(uuid.UUID)
	*req2.TargetID = uuid.New()
	req2.CensusID = ""
	// make request
	resp2 := wsc.Request(req2, s)
	// check register went successful
	if resp2.Ok {
		t.Fatal("should fail if len(censusId) == 0")
	}

	// should fail if cannot decode censusID
	var req3 types.MetaRequest
	req3.Method = "addCensus"
	req3.TargetID = new(uuid.UUID)
	*req3.TargetID = uuid.New()
	req3.CensusID = ""
	// make request
	resp3 := wsc.Request(req3, s)
	// check register went successful
	if resp3.Ok {
		t.Fatal("should fail if cannot decode censusId")
	}

	// TODO  Enable when targets implemented
	// should fail if db Target() fails
	// var req4 types.MetaRequest
	// s2 := ethereum.NewSignKeys()
	// s2.AddHexKey(testdb.Signers[3].Priv)
	// req4.Method = "addCensus"
	// req4.TargetID = uuid.New()
	// req4.CensusID = "d67fb28849af7543f2b0b6bf01bde17613bf7ada"
	// // make request
	// resp4 := wsc.Request(req4, s2)
	// // check register went successful
	// if resp4.Ok {
	// 	t.Fatal("should fail if db Target() fails")
	// }

	// should fail if addCensus fails
	var req5 types.MetaRequest
	s3 := ethereum.NewSignKeys()
	s3.AddHexKey(testdb.Signers[2].Priv)
	req5.Method = "addCensus"
	req5.TargetID = new(uuid.UUID)
	*req5.TargetID = uuid.New()
	req5.CensusID = "d67fb28849af7543f2b0b6bf01bde17613bf7ada"
	// make request
	resp5 := wsc.Request(req5, s3)
	// check register went successful
	if resp5.Ok {
		t.Fatal("should fail if db AddCensus() fails")
	}

	// otherwise should success
	var req6 types.MetaRequest
	req6.Method = "addCensus"
	req6.TargetID = new(uuid.UUID)
	*req6.TargetID = uuid.New()
	req6.CensusID = "d67fb28849af7543f2b0b6bf01bde17613bf7ada"
	// make request
	resp6 := wsc.Request(req6, s)
	// check register went successful
	if !resp6.Ok {
		t.Fatal("should success")
	}
}

func TestGetCensus(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}
	// should fail if len(censusID) == 0
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[0].Priv)
	var req types.MetaRequest
	req.Method = "getCensus"
	req.CensusID = ""
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if len(censusID) == 0")
	}

	// should fail if cannot decode census id
	var req2 types.MetaRequest
	req2.Method = "getCensus"
	req2.TargetID = new(uuid.UUID)
	*req2.TargetID = uuid.New()
	req2.CensusID = "0xA"
	// make request
	resp2 := wsc.Request(req2, s)
	// check register went successful
	if resp2.Ok {
		t.Fatal("should fail if cannot decode censusId")
	}

	// should fail if db Census() fails
	var req3 types.MetaRequest
	req3.Method = "getCensus"
	req3.CensusID = "d67fb28849af7543f2b0b6bf01bde17613bf7ada"
	// make request
	resp3 := wsc.Request(req3, s)
	// check register went successful
	if resp3.Ok {
		t.Fatal("should fail if db Census() fails")
	}

	// otherwise should success
	var req4 types.MetaRequest
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[2].Priv)
	req4.Method = "getCensus"
	req4.CensusID = "d67fb28849af7543f2b0b6bf01bde17613bf7ada"
	// make request
	resp4 := wsc.Request(req4, s2)
	// check register went successful
	if !resp4.Ok {
		t.Fatal("should success")
	}
}

func TestCountCensus(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}
	// should fail if db CountCensus() fails
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[1].Priv)
	var req types.MetaRequest
	req.Method = "countCensus"
	req.CensusID = "d67fb28849af7543f2b0b6bf01bde17613bf7ada"
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if CountCensus() fails")
	}

	// otherwise should success
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[0].Priv)
	var req2 types.MetaRequest
	req2.Method = "countCensus"
	req2.CensusID = "d67fb28849af7543f2b0b6bf01bde17613bf7ada"
	// make request
	resp2 := wsc.Request(req2, s2)
	// check register went successful
	if !resp2.Ok {
		t.Fatal("should success")
	}
}

func TestListCensus(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}
	// should fail if checkOptions fails
	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[1].Priv)
	var req types.MetaRequest
	req.Method = "listCensus"
	req.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "0x",
		Skip:   2,
		SortBy: "lastName",
	}
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if checkOptions fails")
	}

	// should fail if db ListCensus returns no rows
	var req2 types.MetaRequest
	req2.Method = "listCensus"
	req2.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "ascend",
		Skip:   2,
		SortBy: "lastName",
	}
	// make request
	resp2 := wsc.Request(req2, s)
	// check register went successful
	if resp2.Ok {
		t.Fatal("should fail if no rows")
	}
	// should fail if db ListCensus fails
	s2 := ethereum.NewSignKeys()
	s2.AddHexKey(testdb.Signers[0].Priv)
	var req3 types.MetaRequest
	req3.Method = "listCensus"
	req3.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "ascend",
		Skip:   2,
		SortBy: "lastName",
	}
	// make request
	resp3 := wsc.Request(req3, s2)
	// check register went successful
	if resp3.Ok {
		t.Fatal("should fail if db ListCensus fails")
	}

	// otherwise should success
	s3 := ethereum.NewSignKeys()
	s3.AddHexKey(testdb.Signers[2].Priv)
	var req4 types.MetaRequest
	req4.Method = "listCensus"
	req4.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "ascend",
		Skip:   2,
		SortBy: "name",
	}
	// m4ke request
	resp4 := wsc.Request(req4, s3)
	// check register went successful
	if !resp4.Ok {
		t.Fatal("should success")
	}
}

func TestRequestGas(t *testing.T) {
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatal(err)
	}

	s := ethereum.NewSignKeys()
	s.AddHexKey(testdb.Signers[1].Priv)
	var req types.MetaRequest
	req.Method = "requestGas"
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if resp.Ok {
		t.Fatal("should fail if no ethclient provided")
	}
}
