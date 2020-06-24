package testmanager

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
	"gitlab.com/vocdoni/vocdoni-manager-backend/test/testcommon"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

var api testcommon.TestAPI

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	api = testcommon.TestAPI{Port: 12000 + rand.Intn(1000)}
	db := &config.DB{
		Dbname:   "vocdonimgr",
		Password: "vocdoni",
		Host:     "127.0.0.1",
		Port:     5432,
		Sslmode:  "disable",
		User:     "vocdoni",
	}
	api.Start(db, "/api")
	if err := api.DB.Ping(); err != nil {
		log.Printf("SKIPPING: could not connect to DB: %v", err)
		return
	}
	os.Exit(m.Run())
}

func TestSignUp(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	signers, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	req.Method = "signUp"
	resp := wsc.Request(req, signers[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	// cannot add twice
	resp2 := wsc.Request(req, signers[0])
	if resp2.Ok {
		t.Fatal("entity must be unique, cannot add twice")
	}

	if targets, err := api.DB.Targets(entities[0].ID); err != nil || len(targets) != 1 {
		t.Fatal("entities \"all\" automatically created target could not be retrieved")
	}
}

func TestListMembers(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	memInfo := make([]types.Member, len(members))
	for idx, mem := range members {
		memInfo[idx] = *mem
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, memInfo); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "asc",
		Skip:   0,
		SortBy: "lastName",
	}
	// create and make request
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	if len(resp.Members) != 3 {
		t.Fatalf("expected %d members, but got %d", 3, len(resp.Members))
	}
}

func TestGenerateTokens(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into DB: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	randAmount := rand.Intn(100)
	req.Amount = randAmount
	req.Method = "generateTokens"
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}

	if len(resp.Tokens) != randAmount {
		t.Fatalf("expected %d tokens, but got %d", randAmount, len(resp.Tokens))
	}
	// another entity cannot request
}

func TestExportTokens(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	memInfo := make([]types.Member, len(members))
	for idx, mem := range members {
		memInfo[idx] = *mem
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, memInfo); err != nil {
		t.Error(err)
	}

	var req types.MetaRequest
	req.Method = "exportTokens"
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	if len(resp.MembersTokens) != 3 {
		t.Fatalf("expected 3 tokens, but got %d", len(resp.MembersTokens))
	}
	// another entity cannot request
}

func TestGetTarget(t *testing.T) {
	var targetID uuid.UUID
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	inTarget := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	// test adding target
	if targetID, err = api.DB.AddTarget(entities[0].ID, inTarget); err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	req.Method = "getTarget"
	req.TargetID = targetID
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok || resp.Target.Name != "all" {
		t.Fatalf("request failed: %+v", req)
	}

}

func TestListTargets(t *testing.T) {
	var targetID uuid.UUID
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	inTarget := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	// test adding target
	if targetID, err = api.DB.AddTarget(entities[0].ID, inTarget); err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	req.Method = "listTargets"
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok || len(resp.Targets) != 1 || resp.Targets[0].Name != "all" || resp.Targets[0].ID != targetID {
		t.Fatalf("request failed: %+v", req)
	}

}

func TestDumpTarget(t *testing.T) {
	var targetID uuid.UUID
	n := 3 // number of members
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	t.Log(hex.EncodeToString(entities[0].ID))
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, n)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	memInfo := make([]types.Member, len(members))
	for idx, mem := range members {
		memInfo[idx] = *mem
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, memInfo); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	inTarget := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	// test adding target
	if targetID, err = api.DB.AddTarget(entities[0].ID, inTarget); err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	req.Method = "dumpTarget"
	req.TargetID = targetID
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok || len(resp.Claims) != n {
		t.Fatalf("request failed: %+v", req)
	}
	// another entity cannot request
}

func TestImportMembers(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	memInfo := make([]types.MemberInfo, len(members))
	for idx, mem := range members {
		memInfo[idx] = mem.MemberInfo
	}
	// add members
	// if err := api.DB.AddMemberBulk(entities[0].ID, memInfo); err != nil {
	// 	t.Fatalf("cannot add members into database: %s", err)
	// }
	// create and make request
	var req types.MetaRequest
	req.MembersInfo = make([]types.MemberInfo, len(members))
	req.Method = "importMembers"
	for idx, mem := range members {
		req.MembersInfo[idx] = mem.MemberInfo
	}
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	// another entity cannot request
}
