package testmanager

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
	"gitlab.com/vocdoni/vocdoni-manager-backend/test/testcommon"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

var api testcommon.TestAPI

func TestMain(t *testing.M) {
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
	os.Exit(t.Run())
}

func TestSignUp(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Error(err)
	}
	// create entity
	signers, _, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Error(err)
	}

	var req types.MetaRequest
	req.Method = "signUp"
	resp := wsc.Request(req, signers[0])
	if !resp.Ok {
		t.Error()
	}

	// cannot add twice
	resp2 := wsc.Request(req, signers[0])
	if resp2.Ok {
		t.Error()
	}
}

// TBD: fail on inner type reflect
func TestListMembers(t *testing.T) {
	db := api.DB
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Error(err)
	}
	// create entity
	entitySigners, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Error(err)
	}
	// add entity
	db.AddEntity(entities[0].ID, &entities[0].EntityInfo)

	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	memInfo := make([]types.MemberInfo, len(members))
	for idx, mem := range members {
		memInfo[idx] = mem.MemberInfo
	}
	// add members
	if err := db.AddMemberBulk(entities[0].ID, memInfo); err != nil {
		t.Error(err)
	}

	var req types.MetaRequest
	req.Method = "listMembers"
	req.EntityID = hex.EncodeToString(entities[0].ID)
	req.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "asc",
		Skip:   0,
		SortBy: "lastName",
	}

	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Error()
	}
	if len(resp.Members) != 3 {
		t.Error()
	}
}

func TestGenerateTokens(t *testing.T) {
	db := api.DB
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Error(err)
	}
	// create entity
	entitySigners, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Error(err)
	}
	// add entity
	db.AddEntity(entities[0].ID, &entities[0].EntityInfo)

	var req types.MetaRequest
	randAmount := rand.Intn(100)
	req.Amount = randAmount
	req.Method = "generateTokens"

	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Error()
	}
	if len(resp.MembersTokens) != randAmount {
		t.Error("amounts do not match")
	}

	// another entity cannot request
}

func TestExportTokens(t *testing.T) {
	db := api.DB
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Error(err)
	}
	// create entity
	entitySigners, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Error(err)
	}
	// add entity
	db.AddEntity(entities[0].ID, &entities[0].EntityInfo)

	var req types.MetaRequest
	randAmount := rand.Intn(100)
	req.Amount = randAmount
	req.Method = "exportTokens"

	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Error()
	}
	if len(resp.MembersTokens) != randAmount {
		t.Error("amounts do not match")
	}
	// another entity cannot request
}

func TestImportMembers(t *testing.T) {
	db := api.DB
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Error(err)
	}
	// create entity
	entitySigners, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Error(err)
	}
	// add entity
	db.AddEntity(entities[0].ID, &entities[0].EntityInfo)

	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	memInfo := make([]types.MemberInfo, len(members))
	for idx, mem := range members {
		memInfo[idx] = mem.MemberInfo
	}
	// add members
	if err := db.AddMemberBulk(entities[0].ID, memInfo); err != nil {
		t.Error(err)
	}

	var req types.MetaRequest
	req.Method = "importMembers"
	for idx, mem := range members {
		req.MembersInfo[idx] = mem.MemberInfo
	}

	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Error()
	}

	// another entity cannot request
}
