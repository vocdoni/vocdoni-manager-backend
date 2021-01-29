package testregistry

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	qt "github.com/frankban/quicktest"
	"github.com/google/uuid"
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

func TestAuthentication(t *testing.T) {
	c := qt.New(t)
	// create entity
	_, entities := testcommon.CreateEntities(1)
	entities[0].CallbackSecret = "test"
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into DB: %s", err)
	}

	membersUids, err := api.DB.CreateNMembers(entities[0].ID, 2)
	if err != nil {
		t.Fatalf("cannot create uuids: %s", err)
	}
	// connect to endpoint
	wsc, err := testcommon.NewHTTPapiConnection(fmt.Sprintf("http://127.0.0.1:%d/api/token", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}

	// 1. Correct request should succeed
	var req types.MetaRequest
	req.Method = "status"
	req.EntityID = entities[0].ID
	req.Token = membersUids[0].String()
	req.Timestamp = int32(time.Now().Unix())
	req.AuthHash = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)

	resp := wsc.Request(req, nil)
	if !resp.Ok {
		t.Fatalf("correct request did not succeed: %+v", resp)
	}

	// 2. Request with old timestamp should fail
	req.Token = membersUids[0].String()
	req.Timestamp = int32(time.Now().Unix()) - 4000
	req.AuthHash = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)

	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("request with timestamp older than 4s should  fail: %+v", resp)
	}

	// 3. Request with future timestamp should fail
	req.Timestamp = int32(time.Now().Unix()) + 4000
	req.AuthHash = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token)
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("request with timestamp in the future (4s) should  fail: %+v", resp)
	}

	// 4. Request generated with wrong secret should fail
	req.Timestamp = int32(time.Now().Unix())
	req.AuthHash = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, "wrong")
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("request with wrong secret did not fail: %+v", resp)
	}

	// 5. Request generated with wrong secret should fail
	req.Timestamp = int32(time.Now().Unix())
	req.AuthHash = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp+10), req.Token, entities[0].CallbackSecret)
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("request with auth calculated with different entries than request did not fail: %+v", resp)
	}

	// 6. Request without authHash field gshould fail
	req.Timestamp = int32(time.Now().Unix())
	req.AuthHash = ""
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("request with auth calculated with different entries than request did not fail: %+v", resp)
	}

	err = api.DB.DeleteEntity(entities[0].ID)
	c.Check(err, qt.IsNil)
}

func TestGenerate(t *testing.T) {
	c := qt.New(t)
	// connect to endpoint
	wsc, err := testcommon.NewHTTPapiConnection(fmt.Sprintf("http://127.0.0.1:%d/api/token", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	_, entities := testcommon.CreateEntities(1)
	entities[0].CallbackSecret = "test"
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into DB: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	randAmount := rand.Intn(100)
	req.EntityID = entities[0].ID
	req.Amount = randAmount
	req.Method = "generate"
	req.Timestamp = int32(time.Now().Unix())
	auth := calculateAuth(fmt.Sprintf("%d", req.Amount), req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret)
	req.AuthHash = auth
	resp := wsc.Request(req, nil)
	if !resp.Ok {
		t.Fatalf("request failed: %+v", resp)
	}

	if len(resp.Tokens) != randAmount {
		t.Fatalf("expected %d tokens, but got %d", randAmount, len(resp.Tokens))
	}

	// another entity cannot request

	err = api.DB.DeleteEntity(entities[0].ID)
	c.Check(err, qt.IsNil)
}

func TestStatus(t *testing.T) {
	c := qt.New(t)
	// create entity
	_, entities := testcommon.CreateEntities(1)
	entities[0].CallbackSecret = "test"
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into DB: %s", err)
	}

	membersUids, err := api.DB.CreateNMembers(entities[0].ID, 2)
	if err != nil {
		t.Fatalf("cannot create uuids: %s", err)
	}
	// connect to endpoint
	wsc, err := testcommon.NewHTTPapiConnection(fmt.Sprintf("http://127.0.0.1:%d/api/token", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// 0. Existing unused uuid should return available
	var req types.MetaRequest
	req.Method = "status"
	req.EntityID = entities[0].ID
	req.Token = membersUids[0].String()
	req.Timestamp = int32(time.Now().Unix())
	auth := calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)
	req.AuthHash = auth
	resp := wsc.Request(req, nil)
	c.Assert(resp.Ok, qt.IsTrue)
	c.Assert(resp.TokenStatus, qt.Equals, "available", qt.Commentf("expected token \"available\", but got %s", resp.TokenStatus))

	// 1. Existing used uuid should return registered
	s := new(ethereum.SignKeys)
	s.Generate()
	pub := s.PublicKey()

	if err = api.DB.RegisterMember(entities[0].ID, pub, &membersUids[0]); err != nil {
		t.Fatalf("unable to register member: (%v)", err)
	}

	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	c.Assert(resp.Ok, qt.IsTrue)
	c.Assert(resp.TokenStatus, qt.Equals, "registered", qt.Commentf("expected token \"registered\", but got %s", resp.TokenStatus))

	// 2. Non-Existing  uuid should return invalid
	req.Token = uuid.New().String()
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	c.Assert(resp.Ok, qt.IsTrue)
	c.Assert(resp.TokenStatus, qt.Equals, "invalid", qt.Commentf("expected token \"invalid\", but got %s", resp.TokenStatus))

	// 3. Valid ID and Non-Existing  entity should return error invalid entity Id
	req.Token = membersUids[1].String()
	req.EntityID = util.RandomBytes(ethcommon.AddressLength)
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	c.Assert(resp.Ok, qt.IsFalse, qt.Commentf("expected  error message \"invalid entityID\", but request succeeded: %+v", resp))
	c.Assert(resp.Message, qt.Equals, "invalid entityID", qt.Commentf("expected  error message \"invalid entityID\", but got %s", resp.Message))

	// 4. Invalid ID and Non-Existing  entity should return error invalid entity Id
	req.Token = membersUids[1].String()
	req.EntityID = util.RandomBytes(ethcommon.AddressLength)
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	c.Assert(resp.Ok, qt.IsFalse, qt.Commentf("expected  error message \"invalid entityID\", but request succeeded: %+v", resp))
	c.Assert(resp.Message, qt.Equals, "invalid entityID", qt.Commentf("expected  error message \"invalid entityID\", but got %s", resp.Message))

	// 5. Valid ID and Empty  entity should return error invalid entity Id
	req.Token = membersUids[1].String()
	req.EntityID = nil
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	c.Assert(resp.Ok, qt.IsFalse, qt.Commentf("expected  error message \"invalid entityID\", but request succeeded: %+v", resp))
	c.Assert(resp.Message, qt.Equals, "invalid entityID", qt.Commentf("expected  error message \"invalid entityID\", but got %s", resp.Message))

	// 6. Empty ID and Valid entity should return error invalid invalid token
	req.Token = ""
	req.EntityID = entities[0].ID
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	c.Assert(resp.Ok, qt.IsFalse, qt.Commentf("expected  error message \"invalid token\", but request succeeded: %+v", resp))
	c.Assert(resp.Message, qt.Equals, "invalid token", qt.Commentf("expected  error message \"invalid token\", but got %s", resp.Message))

	err = api.DB.DeleteEntity(entities[0].ID)
	c.Check(err, qt.IsNil)
}

func TestRevoke(t *testing.T) {
	c := qt.New(t)
	// create entity
	_, entities := testcommon.CreateEntities(1)
	entities[0].CallbackSecret = "test"
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into DB: %s", err)
	}

	membersUids, err := api.DB.CreateNMembers(entities[0].ID, 2)
	if err != nil {
		t.Fatalf("cannot create uuids: %s", err)
	}
	// connect to endpoint
	wsc, err := testcommon.NewHTTPapiConnection(fmt.Sprintf("http://127.0.0.1:%d/api/token", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// 0. Existing uuid should return get revoked successfully
	var req types.MetaRequest
	req.Method = "revoke"
	req.EntityID = entities[0].ID
	req.Token = membersUids[0].String()
	req.Timestamp = int32(time.Now().Unix())
	auth := calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)
	req.AuthHash = auth
	resp := wsc.Request(req, nil)
	c.Assert(resp.Ok, qt.IsTrue)

	if member, err := api.DB.Member(entities[0].ID, &membersUids[0]); err != nil {
		if err != sql.ErrNoRows {
			//expected
			t.Fatalf("failed retrieving member from db: %+v", resp)
		}
	} else if member != nil {
		t.Fatalf("token (%q) was not revoked", req.Token)
	}

	// 1. Revoking twice the same token should return an error
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	c.Assert(resp.Ok, qt.IsFalse, qt.Commentf("succeed to revoke twice the same token %s : %v", req.Token, resp))

	// 2. Non- Existing uuid should return error
	req.Token = uuid.New().String()
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	c.Assert(resp.Ok, qt.IsFalse, qt.Commentf("succeed to revoke a non-existing token: %v", resp))

	// 3. Valid ID and Non-Existing  entity should return invalid entity Id
	req.Token = membersUids[1].String()
	req.EntityID = util.RandomBytes(ethcommon.AddressLength)
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	c.Assert(resp.Ok, qt.IsFalse, qt.Commentf("expected  error message \"invalid entityID\", but request succeeded: %+v", resp))
	c.Assert(resp.Message, qt.Equals, "invalid entityID", qt.Commentf("expected  error message \"invalid entityID\", but got %s", resp.Message))

	err = api.DB.DeleteEntity(entities[0].ID)
	c.Check(err, qt.IsNil)

}

func TestImportKeysBulk(t *testing.T) {
	c := qt.New(t)
	// create entity
	_, entities := testcommon.CreateEntities(1)
	entities[0].CallbackSecret = "test"
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into DB: %s", err)
	}

	// create keys
	keys := make([][]byte, 100)
	keysString := make([]string, 100)
	bulkSigner := ethereum.NewSignKeys()
	for i := range keys {
		if err := bulkSigner.Generate(); err != nil {
			t.Fatalf("error generating ethereum keys: (%v)", err)
		}
		pubBytes := bulkSigner.PublicKey()
		keys[i] = pubBytes
		keysString[i] = fmt.Sprintf("%x", pubBytes)
	}
	// connect to endpoint
	wsc, err := testcommon.NewHTTPapiConnection(fmt.Sprintf("http://127.0.0.1:%d/api/token", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// 0. Importing new keys should succeed
	var req types.MetaRequest
	req.Method = "importKeysBulk"
	req.Keys = keysString
	req.EntityID = entities[0].ID
	req.Timestamp = int32(time.Now().Unix())
	auth := calculateAuth(req.Keys, req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret)
	req.AuthHash = auth
	resp := wsc.Request(req, nil)
	if !resp.Ok {
		t.Errorf("request failed: %+v", resp)
	}

	// verify  added users and members
	for _, claim := range keys {
		user, err := api.DB.User(claim)
		if err != nil || user == nil {
			t.Errorf("could not retrieve user added using importKeysBulk: (%v)", err)
		}
		member, err := api.DB.MemberPubKey(entities[0].ID, claim)
		if err != nil || member == nil {
			t.Errorf("could not retrieve member added using importKeysBulk: (%v)", err)
		}
	}

	// 1. repeated keys should fail
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.Keys, req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Errorf("succeeded to import duplicate keys: %+v", resp)
	}

	err = api.DB.DeleteEntity(entities[0].ID)
	c.Check(err, qt.IsNil)
}

func TestListKeys(t *testing.T) {
	c := qt.New(t)
	// connect to endpoint
	wsc, err := testcommon.NewHTTPapiConnection(fmt.Sprintf("http://127.0.0.1:%d/api/token", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	entities[0].CallbackSecret = "test"
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// create members for
	_, members, err := testcommon.CreateMembers(entities[0].ID, 10)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, members); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	// 1. Test request with default values
	var req types.MetaRequest
	req.Method = "listKeys"
	req.ListOptions = &types.ListOptions{}
	req.EntityID = entities[0].ID
	req.Timestamp = int32(time.Now().Unix())
	auth := calculateAuth(req.EntityID, req.ListOptions, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret)
	req.AuthHash = auth
	// create and make request
	resp := wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsTrue)
	c.Assert(resp.Keys, qt.HasLen, 10)

	//2. Test request with ListOptions
	req.ListOptions = &types.ListOptions{
		Count: 10,
		Skip:  2,
	}
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.ListOptions, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsTrue)
	c.Assert(resp.Keys, qt.HasLen, 8)

	//3.  check sqli guard (protection against sqli)
	req.ListOptions = &types.ListOptions{
		Order: "ascend",
	}
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.ListOptions, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsFalse)

	req.ListOptions = &types.ListOptions{
		SortBy: " ",
	}
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.ListOptions, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsFalse)

	req.ListOptions = &types.ListOptions{
		Order:  "ascend",
		SortBy: "(case/**/when/**/1=1/**/then/**/email/**/else/**/phone/**/end);",
	}
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.ListOptions, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsFalse)

	err = api.DB.DeleteEntity(entities[0].ID)
	c.Check(err, qt.IsNil)

}

func TestDeleteKeys(t *testing.T) {
	c := qt.New(t)
	// connect to endpoint
	wsc, err := testcommon.NewHTTPapiConnection(fmt.Sprintf("http://127.0.0.1:%d/api/token", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(2)
	entities[0].CallbackSecret = "test"
	entities[1].CallbackSecret = "test"
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	if err := api.DB.AddEntity(entities[1].ID, &entities[1].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// create members for
	_, members, err := testcommon.CreateMembers(entities[0].ID, 7)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	membersKeys := make([]string, len(members))
	for i, mem := range members {
		membersKeys[i] = fmt.Sprintf("%x", mem.PubKey)
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, members); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	// 1. Test request with default values
	var req types.MetaRequest
	req.Method = "deleteKeys"
	req.Keys = membersKeys[:5]
	req.EntityID = entities[0].ID
	req.Timestamp = int32(time.Now().Unix())
	auth := calculateAuth(req.EntityID, req.Keys, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret)
	req.AuthHash = auth
	// create and make request
	resp := wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsTrue)
	c.Assert(resp.Count, qt.Equals, 5)
	c.Assert(resp.InvalidKeys, qt.HasLen, 0)

	//2. Test invalid keys (using the same keys as before)
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.Keys, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsTrue)
	c.Assert(resp.Count, qt.Equals, 0)
	c.Assert(resp.InvalidKeys, qt.ContentEquals, req.Keys)

	//3. Test Duplicates
	req.Keys = []string{membersKeys[5], membersKeys[5]}
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth(req.EntityID, req.Keys, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret)
	req.AuthHash = auth
	resp = wsc.Request(req, entitySigners[0])
	c.Assert(resp.Ok, qt.IsTrue)
	c.Assert(resp.Count, qt.Equals, 1)
	c.Assert(resp.InvalidKeys, qt.HasLen, 0)

	err = api.DB.DeleteEntity(entities[0].ID)
	c.Check(err, qt.IsNil)
}

func calculateAuth(fields ...interface{}) string {

	if len(fields) == 0 {
		return ""
	}
	var toHash bytes.Buffer
	for _, f := range fields {
		switch v := f.(type) {
		case string:
			toHash.WriteString(v)
		case []string:
			for _, key := range v {
				toHash.WriteString(key)
			}
		case types.ListOptions:
			toHash.WriteString(fmt.Sprintf("%d%d%s%s", v.Skip, v.Count, v.Order, v.SortBy))
		case []byte:
			toHash.Write(v)
		case types.HexBytes:
			toHash.Write(v)
		}
	}
	return hex.EncodeToString(ethereum.HashRaw(toHash.Bytes()))
}
