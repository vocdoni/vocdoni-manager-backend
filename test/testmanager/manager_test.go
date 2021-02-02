package testmanager

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	qt "github.com/frankban/quicktest"
	"github.com/google/uuid"
	"go.vocdoni.io/dvote/crypto"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/util"
	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/test/testcommon"
	"go.vocdoni.io/manager/types"
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
	if err := api.Start(db, "/api"); err != nil {
		log.Printf("SKIPPING: could not start the API: %v", err)
		return
	}
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
	signers, entities := testcommon.CreateEntities(2)
	// create and make simple request
	var req types.MetaRequest
	req.Method = "signUp"
	resp := wsc.Request(req, signers[0])
	if !resp.Ok {
		t.Fatalf("entity singup without data failed: %+v", req)
	}
	// cannot add twice
	resp2 := wsc.Request(req, signers[0])
	if resp2.Ok {
		t.Fatal("entity must be unique, cannot add twice")
	}

	if targets, err := api.DB.ListTargets(entities[0].ID); err != nil || len(targets) != 1 {
		t.Fatal("entities \"all\" automatically created target could not be retrieved")
	}

	// verify that information gets stored correctly
	req.Method = "signUp"
	req.Entity = &types.EntityInfo{}
	req.Entity.Name = entities[1].Name
	req.Entity.Email = entities[1].Email
	resp = wsc.Request(req, signers[1])
	if !resp.Ok {
		t.Fatalf("entity singUp with data failed: %+v", req)
	}

	entity, err := api.DB.Entity(entities[1].ID)
	if err != nil {
		t.Fatal("error retrieving entity after signUp")
	}
	if entity.Name != entities[1].Name || entity.Email != entities[1].Email {
		t.Fatalf("entity signUp data were not stored correctly")
	}
}

func TestGetEntity(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(2)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// create and make request
	var req types.MetaRequest
	req.Method = "getEntity"
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("failed to get Entity: %+v", req)
	}
	if string(resp.Entity.ID) != string(entities[0].ID) {
		t.Fatalf("retrieved wrong entity")
	}

	// fails for inexisting entity
	resp = wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("managed to get inexisting Entity: %+v", req)
	}
}

func TestUpdateEntity(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// Create and add entity
	signers, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &types.EntityInfo{CensusManagersAddresses: entities[0].CensusManagersAddresses}); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// update without data should fail
	var req types.MetaRequest
	req.Method = "updateEntity"
	resp := wsc.Request(req, signers[0])
	if resp.Ok {
		t.Fatalf("entity update without data succeeded: %+v", req)
	}

	// update with correct data should succeed
	req.Entity = &types.EntityInfo{
		Name:  entities[0].Name,
		Email: entities[0].Email,
	}
	resp = wsc.Request(req, signers[0])
	if !resp.Ok {
		t.Fatalf("entity update with data failed: %+v", req)
	}

	entity, err := api.DB.Entity(entities[0].ID)
	if err != nil {
		t.Fatal("error retrieving entity after signUp")
	}
	if entity.Name != entities[0].Name || entity.Email != entities[0].Email {
		t.Fatalf("entity data were not updated correctly")
	}

	// should not update data that are not allowed to be updated
	censusManagersAddresses := util.RandomBytes(ethcommon.AddressLength)
	req.Entity = &types.EntityInfo{
		Name:                    "New",
		Email:                   "New",
		CensusManagersAddresses: [][]byte{censusManagersAddresses},
	}
	resp = wsc.Request(req, signers[0])
	if !resp.Ok {
		t.Fatalf("entity update with data failed: %+v", req)
	}

	entity, err = api.DB.Entity(entities[0].ID)
	if err != nil {
		t.Fatal("error retrieving entity after signUp")
	}
	if bytes.Equal(entity.CensusManagersAddresses[0], censusManagersAddresses) {
		t.Fatalf("entity data were updated while they should not")
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
	entitySigners, entities := testcommon.CreateEntities(3)
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
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, members); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "ascend",
		Skip:   2,
		SortBy: "lastName",
	}
	// create and make request
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	if len(resp.Members) != 1 {
		t.Fatalf("expected %d members, but got %d", 1, len(resp.Members))
	}

	// check members are returned ordered
	// add entity
	if err := api.DB.AddEntity(entities[1].ID, &entities[1].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// add members
	// create members
	_, members, err = testcommon.CreateMembers(entities[1].ID, 10)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[1].ID, members); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "ascend",
		Skip:   0,
		SortBy: "firstName",
	}
	// create and make request
	resp = wsc.Request(req, entitySigners[1])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	if len(resp.Members) != 10 {
		t.Fatalf("expected %d members, but got %d", 1, len(resp.Members))
	}

	// check sqli guard (protection against sqli)
	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "ascend",
		Skip:   0,
		SortBy: "*",
	}
	// create and make request
	resp = wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}

	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "ascend",
		Skip:   0,
		SortBy: " ",
	}
	// create and make request
	resp = wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}

	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "ascend",
		Skip:   0,
		SortBy: "(case/**/when/**/1=1/**/then/**/email/**/else/**/phone/**/end);",
	}
	// create and make request
	resp = wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}

	t.Logf("members: %+v", resp.Members)
}

func TestGetMember(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 1)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}

	// add member
	var memberID uuid.UUID
	if memberID, err = api.DB.AddMember(entities[0].ID, members[0].PubKey, &members[0].MemberInfo); err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}

	// create and make request
	var req types.MetaRequest
	req.Method = "getMember"
	req.MemberID = &memberID
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok || hex.EncodeToString(resp.Member.PubKey) != hex.EncodeToString(members[0].PubKey) {
		t.Fatalf("request failed: %+v", req)
	}

}

func TestUpdateMember(t *testing.T) {
	c := qt.New(t)
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 2)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}

	// add member
	if members[0].ID, err = api.DB.AddMember(entities[0].ID, members[0].PubKey, &members[0].MemberInfo); err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}
	if members[1].ID, err = api.DB.AddMember(entities[0].ID, members[1].PubKey, &members[1].MemberInfo); err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}

	// newMember := members[0]
	memInfo := members[0].MemberInfo
	newMember := &types.Member{}
	newMember.ID = members[0].ID
	newMember.EntityID = members[0].EntityID
	newMember.MemberInfo = memInfo
	newMember.Email = "upd"
	newMember.FirstName = "upd"
	newMember.LastName = "upd"
	newMember.StreetAddress = ""

	// create and make request
	var req types.MetaRequest
	req.Method = "updateMember"
	req.Member = newMember
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}

	var member *types.Member
	if member, err = api.DB.Member(entities[0].ID, &members[0].ID); err != nil {
		t.Fatalf("cannot retrieve udpated member from database: %s", err)
	}
	if member.Email != "upd" || member.FirstName != "upd" || member.LastName != "upd" {
		t.Fatalf("cannot update member fields")
	}
	if member.StreetAddress != members[0].StreetAddress {
		t.Fatalf("updating non corresponding fields")
	}

	members[1].Email = "upd"
	req.Member = &members[1]
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsFalse, qt.Commentf("updated member with duplicate email : \n%v\n%v", req, resp))

	err = api.DB.DeleteEntity(entities[0].ID)
	c.Check(err, qt.IsNil, qt.Commentf("error cleaning up"))
}

func TestDeleteMembers(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 4)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}

	// add members
	if members[0].ID, err = api.DB.AddMember(entities[0].ID, members[0].PubKey, &members[0].MemberInfo); err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}
	if members[1].ID, err = api.DB.AddMember(entities[0].ID, members[1].PubKey, &members[1].MemberInfo); err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}
	if members[2].ID, err = api.DB.AddMember(entities[0].ID, members[2].PubKey, &members[2].MemberInfo); err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}
	if members[3].ID, err = api.DB.AddMember(entities[0].ID, members[3].PubKey, &members[3].MemberInfo); err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}

	// 1. request without uuids fails
	var req types.MetaRequest
	req.Method = "deleteMembers"
	req.MemberIDs = []uuid.UUID{}
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if resp.Ok {
		t.Fatalf("request succeeded with empty list: %v", req)
	}

	//2. request with a random uuid succeeds but returns invalidID and nothing changes in the DB
	tempUUID := uuid.New()
	req.MemberIDs = []uuid.UUID{members[0].ID, tempUUID}
	resp = wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok {
		t.Fatalf("request succeeded with one random uuid: %v", req)
	}
	if len(resp.InvalidIDs) != 1 || resp.InvalidIDs[0] != tempUUID {
		t.Fatal("invalidID was not returned correctly")
	}

	rows, err := api.DB.CountMembers(entities[0].ID)
	if err != nil {
		t.Fatalf("could retrieve deleted member from database: %s", err)
	}
	if rows != 3 {
		t.Fatalf("expected 3 rows but found %d", rows)
	}

	// if given duplicate members find unique and remove it
	req.MemberIDs = []uuid.UUID{members[0].ID, members[0].ID}
	resp = wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok {
		t.Fatalf("request failed: %v", req)
	}

	rows, err = api.DB.CountMembers(entities[0].ID)
	if err != nil {
		t.Fatalf("could retrieve deleted member from database: %s", err)
	}
	if rows != 3 {
		t.Fatalf("expected 3 rows but found %d", rows)
	}

	// Remove members
	req.MemberIDs = []uuid.UUID{members[1].ID, members[2].ID, members[3].ID}
	resp = wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok {
		t.Fatalf("request failed: %v", req)
	}

	rows, err = api.DB.CountMembers(entities[0].ID)
	if err != nil {
		t.Fatalf("could retrieve deleted member from database: %s", err)
	}
	if rows != 0 {
		t.Fatalf("expected 0 rows but found %d", rows)
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
	entitySigners, entities := testcommon.CreateEntities(1)
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
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	if err != nil {
		t.Fatalf("unable to create testing members: %s", err)
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, members); err != nil {
		t.Error(err)
	}

	// Test that members with public keys are not
	var req types.MetaRequest
	req.Method = "exportTokens"
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	if len(resp.MembersTokens) != 0 {
		t.Fatalf("expected 0 tokens, but got %d", len(resp.MembersTokens))
	}

	// Check that members without public keys are exported
	var importMembers []types.MemberInfo
	for i := 0; i < 10; i++ {
		info := types.MemberInfo{FirstName: fmt.Sprintf("Name%d", i), LastName: fmt.Sprintf("LastName%d", i)}
		importMembers = append(importMembers, info)
	}
	err = api.DB.ImportMembers(entities[0].ID, importMembers)
	if err != nil {
		t.Fatalf("cannot add members to Postgres DB (pgsql.go:importMembers): %s", err)
	}

	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	if len(resp.MembersTokens) != len(importMembers) {
		t.Fatalf("expected %d tokens, but got %d", len(importMembers), len(resp.MembersTokens))
	}

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
	entitySigners, entities := testcommon.CreateEntities(1)
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
	req.TargetID = &targetID
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
	entitySigners, entities := testcommon.CreateEntities(1)
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
	entitySigners, entities := testcommon.CreateEntities(1)
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
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, members); err != nil {
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
	req.TargetID = &targetID
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok || len(resp.Claims) != n {
		t.Fatalf("request failed: %+v", req)
	}
	// another entity cannot request
}

func TestDumpCensus(t *testing.T) {
	c := qt.New(t)
	var targetID uuid.UUID
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// create members
	n := 100
	_, members, _ := testcommon.CreateMembers(entities[0].ID, n)
	memberIDs, err := api.DB.CreateNMembers(entities[0].ID, n)
	if err != nil {
		t.Fatalf("cannot generate random members (%v)", err)
	}

	inTarget := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}
	if targetID, err = api.DB.AddTarget(entities[0].ID, inTarget); err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	id := util.RandomHex(len(entities[0].ID))
	idBytes, err := hex.DecodeString(util.TrimHex(id))
	if err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, &types.CensusInfo{Name: id, Ephemeral: true})
	if err != nil {
		t.Fatalf("cannot add census: (%v)", err)
	}

	// Test only webpoll census
	var req types.MetaRequest
	req.Method = "dumpCensus"
	req.CensusID = id
	resp := wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsTrue, qt.Commentf("request failed: %+v", req))
	c.Assert(resp.Claims, qt.HasLen, n)

	if len(resp.Claims) != n {
		t.Fatalf("expected %d claims but got %d", n, len(resp.Claims))
	}

	census, err := api.DB.Census(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("cannot retrieve census: (%v)", err)
	}
	if !census.Ephemeral {
		t.Fatal("census was marked as non ephemeral while it should")
	}

	ephemeralMembers, err := api.DB.ListEphemeralMemberInfo(entities[0].ID, idBytes)
	c.Assert(err, qt.IsNil, qt.Commentf("testDumpCensus: cannot retrieve ephemeral member info: (%v)", err))
	c.Assert(ephemeralMembers, qt.HasLen, n)

	for _, mem := range ephemeralMembers {
		found := false
		for _, claim := range resp.Claims {
			if fmt.Sprintf("%x", claim) == fmt.Sprintf("%x", mem.DigestedPubKey) {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("emphmeral pubKey not found in claims")
		}
	}

	// Test mix webpoll-app census

	for i := 0; i < n/2; i++ {
		err = api.DB.RegisterMember(entities[0].ID, members[i].PubKey, &memberIDs[i])
		if err != nil {
			t.Fatalf("cannot register member: (%v)", err)
		}
	}

	id = util.RandomHex(len(entities[0].ID))
	idBytes, err = hex.DecodeString(util.TrimHex(id))
	if err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, &types.CensusInfo{Name: id, Ephemeral: true})
	if err != nil {
		t.Fatalf("cannot add census: (%v)", err)
	}

	req.CensusID = id
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsTrue, qt.Commentf("request failed: %+v", req))
	c.Assert(resp.Claims, qt.HasLen, n)

	census, err = api.DB.Census(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("cannot retrieve census: (%v)", err)
	}
	if !census.Ephemeral {
		t.Fatal("census was not marked as ephemeral as it should")
	}

	ephemeralMembers, err = api.DB.ListEphemeralMemberInfo(entities[0].ID, idBytes)
	c.Assert(err, qt.IsNil, qt.Commentf("testDumpCensus: cannot retrieve ephemeral member info: (%v)", err))

	if len(ephemeralMembers) != n/2 {
		t.Fatalf("expected %d emphemeral members but got %d", n/2, len(ephemeralMembers))
	}
	for _, mem := range ephemeralMembers {
		found := false
		for _, claim := range resp.Claims {
			if fmt.Sprintf("%x", claim) == fmt.Sprintf("%x", mem.DigestedPubKey) {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("emphmeral pubKey not found in claims")
		}
	}

	// Test non-webpoll census
	for i := n / 2; i < n; i++ {
		err = api.DB.RegisterMember(entities[0].ID, members[i].PubKey, &memberIDs[i])
		if err != nil {
			t.Fatalf("cannot register member: (%v)", err)
		}
	}

	id = util.RandomHex(len(entities[0].ID))
	idBytes, err = hex.DecodeString(util.TrimHex(id))
	if err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, &types.CensusInfo{Name: id, Ephemeral: false})
	if err != nil {
		t.Fatalf("cannot add census: (%v)", err)
	}

	req.CensusID = id
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsTrue, qt.Commentf("request failed: %+v", req))
	c.Assert(resp.Claims, qt.HasLen, n)

	census, err = api.DB.Census(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("cannot retrieve census: (%v)", err)
	}
	if census.Ephemeral {
		t.Fatal("census was marked as ephemeral while it should not")
	}

	ephemeralMembers, err = api.DB.ListEphemeralMemberInfo(entities[0].ID, idBytes)
	c.Assert(err, qt.IsNil, qt.Commentf("testDumpCensus: cannot retrieve ephemeral member info: (%v)", err))
	c.Assert(ephemeralMembers, qt.HasLen, 0)

	err = api.DB.DeleteEntity(entities[0].ID)
	c.Check(err, qt.IsNil, qt.Commentf("error cleaning up"))

}

func TestSendVotingLinks(t *testing.T) {
	c := qt.New(t)
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(2)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 2)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}

	memInfo := make([]types.MemberInfo, len(members))
	memInfo[0] = members[0].MemberInfo
	memInfo[0].Email = "coby.rippin@ethereal.email"
	memInfo[1] = members[1].MemberInfo
	memInfo[1].Email = "melisa.oberbrunner48@ethereal.email"

	// add members
	if err := api.DB.ImportMembers(entities[0].ID, memInfo); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	dbMembers, err := api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("coulld not retrieve DB members: %v", err)
	}

	// memberIDVerified := dbMembers[0].ID
	memberIDUnverified := dbMembers[1].ID

	inTarget := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}
	targetID, err := api.DB.AddTarget(entities[0].ID, inTarget)
	if err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	processID := util.RandomBytes(ethcommon.HashLength)
	censusID := util.RandomHex(len(entities[0].ID))
	idBytes, err := hex.DecodeString(util.TrimHex(censusID))
	if err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, &types.CensusInfo{Name: censusID, Ephemeral: true})
	if err != nil {
		t.Fatalf("cannot add census: (%v)", err)
	}

	_, err = api.DB.ExpandCensusMembers(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("cannot dump census claims: (%v)", err)
	}
	// Valid request for unverified member should succeed
	var req types.MetaRequest
	req.Method = "sendVotingLinks"
	req.ProcessID = processID
	req.CensusID = censusID
	resp := wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsTrue, qt.Commentf("failed to send validation link to all unverified members: \n%v\n%v", req, resp))
	c.Assert(resp.Count, qt.Equals, 2, qt.Commentf("failed to send validation link to all unverified members: \n%v\n%v", req, resp))

	if err := api.DB.RegisterMember(entities[0].ID, members[0].PubKey, &dbMembers[0].ID); err != nil {
		t.Fatalf("could not register member: %v", err)
	}

	censusID = util.RandomHex(len(entities[0].ID))
	idBytes, err = hex.DecodeString(util.TrimHex(censusID))
	if err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, &types.CensusInfo{Name: censusID, Ephemeral: true})
	if err != nil {
		t.Fatalf("cannot add census: (%v)", err)
	}

	_, err = api.DB.ExpandCensusMembers(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("cannot dump census claims: (%v)", err)
	}

	req.CensusID = censusID
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsTrue, qt.Commentf("failed to send voting link to unverified member : \n%v\n%v", req, resp))
	c.Assert(resp.Count, qt.Equals, 1, qt.Commentf("failed to send voting link to unverified member : \n%v\n%v", req, resp))

	//  verify member tag was added correctly
	memberUnverified, err := api.DB.Member(entities[0].ID, &memberIDUnverified)
	if err != nil {
		t.Fatalf("could not retrieve DB member: %v", err)
	}
	tag, err := api.DB.TagByName(entities[0].ID, "VoteEmailSent")
	if err != nil {
		t.Fatalf("error retrieving tag: (%v)", err)
	}
	if memberUnverified.Tags[0] != tag.ID {
		t.Fatalf("member tags were not updated correctly. Expected to find %d inside %v", tag.ID, memberUnverified.Tags)
	}

	// member with one existing member email
	req.Email = dbMembers[0].Email
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsFalse, qt.Commentf("sent voting link to verified member while it should not : \n%v\n%v", req, resp))
	c.Assert(resp.Count, qt.Equals, 0, qt.Commentf("sent voting link to verified member while it should not : \n%v\n%v", req, resp))

	req.Email = dbMembers[1].Email
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsTrue, qt.Commentf("failed to send voting link to unverified member : \n%v\n%v", req, resp))
	c.Assert(resp.Count, qt.Equals, 1, qt.Commentf("failed to send voting link to unverified member : \n%v\n%v", req, resp))

	// Unverified member request by wrong entity should fail
	resp = wsc.Request(req, entitySigners[1])
	c.Assert(resp.Ok, qt.IsFalse, qt.Commentf("did not fail to send validation link to member of non-existing entity-member combination : \n%v\n%v", req, resp))

	err = api.DB.DeleteEntity(entities[0].ID)
	c.Check(err, qt.IsNil, qt.Commentf("error cleaning up"))
}

func TestImportMembers(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
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

func TestAddCensus(t *testing.T) {
	var req types.MetaRequest
	var censusInfo *types.CensusInfo
	var root, idBytes []byte
	var err error
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	//Add target
	target := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	var targetID uuid.UUID
	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, members); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	// Genreate ID and root
	id := util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode randpom id: %s", err)
	}
	root = util.RandomBytes(32)
	name := fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo = &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}

	//Test simple add census

	req.Method = "addCensus"
	req.CensusID = id
	req.Census = censusInfo
	req.TargetID = &targetID

	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to create a random census: %s", resp.Message)
	}

	//Verify that census exists
	census, err := api.DB.Census(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("unable to recover created census: %s", err)
	}
	if census.Name != name {
		t.Fatal("census stored incorrectly")
	}

	//Test that empty censusID fails
	req.CensusID = ""
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to create a census without censusId: %s", resp.Message)
	}

	// Test that members without keys are not counted
	if _, err = api.DB.CreateNMembers(entities[0].ID, 10); err != nil {
		t.Fatalf("cannot generate random members (%v)", err)
	}

	// Genreate ID and root
	id = util.RandomHex(len(entities[0].ID))
	root = util.RandomBytes(32)
	name = fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo = &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}

	req.Method = "addCensus"
	req.CensusID = id
	req.Census = censusInfo
	req.TargetID = &targetID

	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to create a random census: %s", resp.Message)
	}
}

func TestUpdateCensus(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// Create and add entity
	signers, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &types.EntityInfo{CensusManagersAddresses: entities[0].CensusManagersAddresses}); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	//Add target
	target := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	var targetID uuid.UUID
	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	// Genreate ID and root
	id := util.RandomHex(len(entities[0].ID))
	idBytes, err := hex.DecodeString(util.TrimHex(id))
	if err != nil {
		t.Fatalf("cannot decode randpom id: %s", err)
	}
	root := util.RandomBytes(32)
	name := fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))

	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, &types.CensusInfo{Name: name})
	if err != nil {
		t.Fatal("error adding census to the db")
	}

	// update without data should fail
	var req types.MetaRequest
	req.CensusID = id
	req.Method = "updateCensus"
	resp := wsc.Request(req, signers[0])
	if resp.Ok {
		t.Fatalf("census update without data succeeded: %+v", req)
	}

	// update with correct data should succeed
	req.Census = &types.CensusInfo{
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}

	resp = wsc.Request(req, signers[0])
	if !resp.Ok {
		t.Fatalf("census update with data failed: %+v", req)
	}

	census, err := api.DB.Census(entities[0].ID, idBytes)
	if err != nil {
		t.Fatal("error retrieving census after signUp")
	}
	if fmt.Sprintf("%x", census.MerkleRoot) != fmt.Sprintf("%x", req.Census.MerkleRoot) || census.MerkleTreeURI != req.Census.MerkleTreeURI {
		t.Fatalf("census data were not updated correctly")
	}

	// should not update data that are not allowed to be updated
	req.Census = &types.CensusInfo{
		Ephemeral: true,
	}
	resp = wsc.Request(req, signers[0])
	if !resp.Ok {
		t.Fatalf("census update with data failed: %+v", req)
	}

	census, err = api.DB.Census(entities[0].ID, idBytes)
	if err != nil {
		t.Fatal("error retrieving entity after signUp")
	}
	if census.Ephemeral == true {
		t.Fatalf("entity data were updated while they should not")
	}

}

func TestGetCensus(t *testing.T) {
	var req types.MetaRequest
	var censusInfo *types.CensusInfo
	var root, idBytes []byte
	var err error
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	//Add target
	target := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	var targetID uuid.UUID
	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	// Genreate ID and root
	id := util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode randpom id: %s", err)
	}
	root = util.RandomBytes(32)
	name := fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo = &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}
	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo)
	if err != nil {
		t.Fatal("error adding census to the db")
	}

	//Test simple get census
	req.Method = "getCensus"
	req.CensusID = id

	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to retrieve a random census: %s", resp.Message)
	}
	if resp.Target.ID != targetID {
		t.Fatalf("target from retrieved census does not match original target: %s", resp.Message)
	}

	//Test that empty censusID fails
	req.CensusID = ""
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to retrieve a census without censusId: %s", resp.Message)
	}

	//Test that random censusID fails
	req.CensusID = util.RandomHex(len(entities[0].ID))
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to retrieve a census without censusId: %s", resp.Message)
	}
}

func TestListCensus(t *testing.T) {
	var req types.MetaRequest
	var censusInfo *types.CensusInfo
	var root, idBytes []byte
	var err error
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}

	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	//Test to  get 0 censuses
	req.Method = "listCensus"
	req.ListOptions = new(types.ListOptions)

	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Censuses) != 0 {
		t.Fatalf("error in requesting censuses when no censuses exist: %s", resp.Message)
	}

	//Add target
	target := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	var targetID uuid.UUID
	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	// Genreate ID and root
	id := util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode randpom id: %s", err)
	}
	root = util.RandomBytes(32)
	name := fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo = &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}

	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo)
	if err != nil {
		t.Fatal("error adding census to the db")
	}

	//Test to  get 1 censuses
	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Censuses) != 1 {
		t.Fatalf("error in requesting censuses when 1 census exists: %s", resp.Message)
	}

	//Add second census (needs second target)
	target = &types.Target{EntityID: entities[0].ID, Name: "all1", Filters: json.RawMessage([]byte("{}"))}

	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add second target into database: %s", err)
	}
	id = util.RandomHex(len(entities[0].ID))
	idBytes, err = hex.DecodeString(util.TrimHex(id))
	if err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	censusInfo.Name = fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))

	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo)
	if err != nil {
		t.Fatal("unable to create second census (pgsql.go:AddCensus)")
	}

	//Test simple get census
	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Censuses) != 2 {
		t.Fatalf("unable to list censuses when 2 censuses exist: %s", resp.Message)
	}

	// check order
	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "descend",
		Skip:   0,
		SortBy: "name",
	}

	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Censuses) != 2 {
		t.Fatalf("unable to list censuses when 2 censuses exist: %s", resp.Message)
	}

	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "descend",
		Skip:   1,
		SortBy: "name",
	}

	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Censuses) != 1 {
		t.Fatalf("unable to list censuses when 1 census should be returned: %s", resp.Message)
	}
}

func TestDeleteCensus(t *testing.T) {
	var req types.MetaRequest
	var censusInfo *types.CensusInfo
	var root, idBytes []byte
	var err error
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(2)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	//Add target
	target := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	var targetID uuid.UUID
	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	// Genreate ID and root
	id := util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode randpom id: %s", err)
	}
	root = util.RandomBytes(32)
	name := fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo = &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}

	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo)
	if err != nil {
		t.Fatal("error adding census to the db")
	}

	//Test that census cannot be deleted from another entity
	req.Method = "deleteCensus"
	req.CensusID = id

	resp := wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("able to delete census of another entity: %s", resp.Message)
	}

	//Test simple delete census
	req.Method = "deleteCensus"
	req.CensusID = id

	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to delete census: %s", resp.Message)
	}

	count, err := api.DB.CountCensus(entities[0].ID)
	if err != nil {
		t.Fatalf("error counting censues: %s", err)
	}
	if count != 0 {
		t.Fatalf("expected to count 0 censuses but got %d", count)
	}

	//Test that empty censusID fails
	req.CensusID = ""
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to delete a census without censusId: %s", resp.Message)
	}

	//Test that random censusID fails
	req.CensusID = util.RandomHex(len(entities[0].ID))
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to delete a census with a random censusId: %s", resp.Message)
	}

	//TODO test that if census had members in census_members they were removed
}

func TestSendValidationLinks(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(2)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 2)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}

	memInfo := make([]types.MemberInfo, len(members))
	for idx, mem := range members {
		memInfo[idx] = mem.MemberInfo
	}
	// add members
	if err := api.DB.ImportMembers(entities[0].ID, memInfo); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	dbMembers, err := api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("coulld not retrieve DB members: %v", err)
	}

	if err := api.DB.RegisterMember(entities[0].ID, members[0].PubKey, &dbMembers[0].ID); err != nil {
		t.Fatalf("coulld not register member: %v", err)
	}
	memberIDVerified := dbMembers[0].ID
	memberIDUnverified := dbMembers[1].ID

	// Valid request for unverified member should succeed
	var req types.MetaRequest
	req.Method = "sendValidationLinks"
	req.MemberIDs = []uuid.UUID{memberIDUnverified}
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok || resp.Count != 1 {
		t.Fatalf("failed to send validation link to unverified member: \n%v\n%v", req, resp)
	}

	//  verify member tag was added correctly
	memberUnverified, err := api.DB.Member(entities[0].ID, &memberIDUnverified)
	if err != nil {
		t.Fatalf("could not retrieve DB member: %v", err)
	}
	tag, err := api.DB.TagByName(entities[0].ID, "PendingValidation")
	if err != nil {
		t.Fatalf("error retrieving tag: (%v)", err)
	}
	if memberUnverified.Tags[0] != tag.ID {
		t.Fatalf("member tags were not updated correctly. Expected to find %d inside %v", tag.ID, memberUnverified.Tags)
	}

	// Unverified member request by wrong entity should fail
	resp = wsc.Request(req, entitySigners[1])
	t.Log(resp)
	if resp.Ok {
		t.Fatalf("did not fail to send validation link to member of non-existing entity-member combination : \n%v\n%v", req, resp)
	}

	// Verified member request should fail
	req.MemberIDs = []uuid.UUID{memberIDVerified}
	resp = wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok && resp.Count != 0 {
		t.Fatalf("did not fail to send validation link to verified member : \n%v\n%v", req, resp)
	}

	// Verified member request should fail
	req.MemberIDs = []uuid.UUID{uuid.New()}
	resp = wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok && resp.Count != 0 {
		t.Fatalf("did not fail to send validation link to non-existing member : \n%v\n%v", req, resp)
	}

}

func TestCreateTag(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
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
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, members); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	name := "TestTag"

	//Test simple add tag
	var req types.MetaRequest
	req.Method = "createTag"
	req.TagName = name

	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to create a test tag: %s", resp.Message)
	}

	//Verify that census exists
	tag, err := api.DB.Tag(entities[0].ID, resp.Tag.ID)
	if err != nil {
		t.Fatalf("unable to recover created tag: %s", err)
	}
	if tag.Name != name {
		t.Fatal("tag stored incorrectly")
	}

	//Test that empty tag Name fails
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to create a tag with duplicate name: %s", resp.Message)
	}

	//Test that empty tag Name fails
	req.TagName = ""
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to create a tag with empty tag name: %s", resp.Message)
	}

}

func TestListTags(t *testing.T) {
	var req types.MetaRequest
	var err error
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}

	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	//Test to  get 0 tags
	req.Method = "listTags"
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Tags) != 0 {
		t.Fatalf("error in requesting tags when no tag exists: %s", resp.Message)
	}

	tagID, err := api.DB.AddTag(entities[0].ID, "TestTag")
	if err != nil {
		t.Fatalf("error creating tag:  (%v)", err)
	}

	//Test to  get 1 tag
	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Tags) != 1 || resp.Tags[0].ID != tagID {
		t.Fatalf("error in requesting tags when 1 tag exists: %s", resp.Message)
	}

}

func TestDeleteTag(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(2)
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
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, members); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}
	listedMembers, err := api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	tag := &types.Tag{
		Name: "TestTag",
	}
	tag.ID, err = api.DB.AddTag(entities[0].ID, "TestTag")
	if err != nil {
		t.Fatalf("cannot add tag into database: %s", err)
	}

	// add tag to member
	n, _, err := api.DB.AddTagToMembers(entities[0].ID, []uuid.UUID{listedMembers[0].ID}, tag.ID)
	if err != nil || n != 1 {
		t.Fatalf("failed to add tag into member: %s", err)
	}

	// Test cannnot delete tag of another entity
	var req types.MetaRequest
	req.Method = "deleteTag"
	req.TagID = tag.ID
	resp := wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("able to delete test tag of another entity: %s", resp.Message)
	}

	//Test simple delete tag
	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to delete test tag: %s", resp.Message)
	}

	//Verify that tag was deleted (also from meber info)
	_, err = api.DB.Tag(entities[0].ID, tag.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("able to recover deleted tag: %s", err)
	}

	if member, err := api.DB.Member(entities[0].ID, &(listedMembers[0].ID)); err != nil {
		t.Fatalf("error retrieving existing member: (%v)", err)
	} else if len(member.Tags) > 0 {
		t.Fatal("deleteTag did not remove tag from member")
	}

	//Test that random tag id delete fails
	req.TagID = int32(util.RandomInt(1, 100))
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to delete a random tag: %s", resp.Message)
	}

	//Test that tag id 0 fails
	req.TagID = 0
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("no error return of tag ID 0: %s", resp.Message)
	}

}

func TestAddTag(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(2)
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
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, members); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}
	listedMembers, err := api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}
	memberIDs := make([]uuid.UUID, len(listedMembers))
	for i, member := range listedMembers {
		memberIDs[i] = member.ID
	}

	tag := &types.Tag{
		Name: "TestTag",
	}
	tag.ID, err = api.DB.AddTag(entities[0].ID, "TestTag")
	if err != nil {
		t.Fatalf("cannot add tag into database: %s", err)
	}

	// Test add tags to members
	var req types.MetaRequest
	req.Method = "addTag"
	req.TagID = tag.ID
	req.MemberIDs = memberIDs
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to add test tag to members: %s", resp.Message)
	}

	if resp.Count != len(memberIDs) || len(resp.InvalidIDs) != 0 {
		t.Fatalf("expected to receive an updated count of %d but received %d", len(memberIDs), resp.Count)
	}

	listedMembers, err = api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}
	for _, member := range listedMembers {
		if member.Tags[0] != tag.ID {
			t.Fatalf("unable to update correctly tag to members")
		}
	}

	// Test that addning same members twice returns them as invalidIDs
	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to add test tag test members: %s", resp.Message)
	}
	if resp.Count != 0 || len(resp.InvalidIDs) != len(memberIDs) {
		t.Fatal("unexpected response for adding for second time  tag from memberIDs")
	}

	//Test fails for wrong entity tag
	resp = wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("able to add tag to another entity: %s", resp.Message)
	}

	//Test fails for random tag
	req.TagID = int32(util.RandomInt(10, 100))
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to add non-existing tag: %s", resp.Message)
	}

	//Test fails for 0 tag
	req.TagID = 0
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to add 0 tag: %s", resp.Message)
	}

	//Test returns the invalidIDs
	tempUUID := uuid.New()
	req.TagID = tag.ID
	req.MemberIDs = []uuid.UUID{tempUUID}
	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to add test tag to members: %s", resp.Message)
	}
	if len(resp.InvalidIDs) != 1 || resp.InvalidIDs[0] != tempUUID {
		t.Fatal("returns incorrectly the InvalidIDs")
	}

}

func TestRemoveTag(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(2)
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
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, members); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}
	listedMembers, err := api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}
	memberIDs := make([]uuid.UUID, len(listedMembers))
	for i, member := range listedMembers {
		memberIDs[i] = member.ID
	}

	tag := &types.Tag{
		Name: "TestTag",
	}
	tag.ID, err = api.DB.AddTag(entities[0].ID, "TestTag")
	if err != nil {
		t.Fatalf("cannot add tag into database: %s", err)
	}
	// add tag to members
	if n, _, err := api.DB.AddTagToMembers(entities[0].ID, memberIDs, tag.ID); err != nil || n != len(memberIDs) {
		t.Fatalf("failed to add tag to members: %s", err)
	}

	var req types.MetaRequest
	req.Method = "removeTag"
	req.TagID = tag.ID
	req.MemberIDs = memberIDs
	//Test fails for wrong entity tag
	resp := wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("able to remove tag to another entity: %s", resp.Message)
	}

	//Test fails for random tag
	req.TagID = int32(util.RandomInt(10, 100))
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to remove non-existing tag: %s", resp.Message)
	}

	//Test fails for 0 tag
	req.TagID = 0
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to remove 0 tag: %s", resp.Message)
	}

	//Test fails for empty memberlist
	req.TagID = tag.ID
	req.MemberIDs = nil
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to remove tag for nil member list: %s", resp.Message)
	}

	// Test fails if non-existing ID is not returned as invalidID
	tempUUID := uuid.New()
	req.MemberIDs = []uuid.UUID{tempUUID}
	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to remove test tag test members: %s", resp.Message)
	}

	if resp.Count != 0 || len(resp.InvalidIDs) != 1 || resp.InvalidIDs[0] != tempUUID {
		t.Fatal("unexpected response for non existing memberID")
	}

	// Test succeeds (with duplicates)
	memberIDs = append(memberIDs, memberIDs[0])
	req.TagID = tag.ID
	req.MemberIDs = memberIDs
	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to remove test tag test members: %s", resp.Message)
	}

	if resp.Count != len(memberIDs)-1 || len(resp.InvalidIDs) != 0 {
		t.Fatal("unexpected response for removing tag from memberIDs with duplicate id")
	}

	listedMembers, err = api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("cannot list members from database: %s", err)
	}
	for _, member := range listedMembers {
		if len(member.Tags) > 0 {
			t.Fatalf("unable to remove correctly tag from members")
		}
	}

	// Test that removing same members twice returns them as invalidIDs
	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to remove test tag test members: %s", resp.Message)
	}
	if resp.Count != 0 || len(resp.InvalidIDs) != len(memberIDs)-1 {
		t.Fatal("unexpected response for removing for second time  tag from memberIDs")
	}

}

func TestDvoteJSSignature(t *testing.T) {
	signer := ethereum.NewSignKeys()
	signer.AddHexKey("c6446f24d08a34fdefc2501d6177b25e8a1d0f589b7a06f5a0131e9a8d0307e4")
	test := struct {
		A string `json:"a"`
	}{
		A: "1",
	}
	a, err := crypto.SortedMarshalJSON(test)
	if err != nil {
		t.Fatalf("%v", err)
	}
	signature, err := signer.Sign(a)
	if err != nil {
		t.Fatalf("%v", err)
	}
	expectedSignature := "361d97d64186bc85cf41d918c9f4bb4ffa08cd756cfb57ab9fe2508808eabfdd5ab16092e419bb17840db104f07ee5452e0551ba61aa6b458e177bae224ee5ad00"
	if fmt.Sprintf("%x", signature) != expectedSignature {
		t.Fatalf("expected signature %s but got %s", expectedSignature, signature)
	}
}
