package testregistry

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/manager/manager-backend/config"
	"gitlab.com/vocdoni/manager/manager-backend/test/testcommon"
	"gitlab.com/vocdoni/manager/manager-backend/types"
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
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/token", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}

	// 1. Correct request should succeed
	var req types.MetaRequest
	req.Method = "status"
	req.EntityID = fmt.Sprintf("%x", entities[0].ID)
	req.Token = membersUids[0].String()
	req.Timestamp = int32(time.Now().Unix())
	req.AuthHash = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})

	resp := wsc.Request(req, nil)
	if !resp.Ok {
		t.Fatalf("correct request did not succeed: %+v", resp)
	}

	// 2. Request with old timestamp should fail
	req.Token = membersUids[0].String()
	req.Timestamp = int32(time.Now().Unix()) - 4000
	req.AuthHash = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})

	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("request with timestamp older than 4s should  fail: %+v", resp)
	}

	// 3. Request with future timestamp should fail
	req.Timestamp = int32(time.Now().Unix()) + 4000
	req.AuthHash = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token})
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("request with timestamp in the future (4s) should  fail: %+v", resp)
	}

	// 4. Request generated with wrong secret should fail
	req.Timestamp = int32(time.Now().Unix())
	req.AuthHash = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, "wrong"})
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("request with wrong secret did not fail: %+v", resp)
	}

	// 5. Request generated with wrong secret should fail
	req.Timestamp = int32(time.Now().Unix())
	req.AuthHash = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp+10), req.Token, entities[0].CallbackSecret})
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

}

func TestGenerate(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/token", api.Port), t)
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
	req.EntityID = fmt.Sprintf("%x", entities[0].ID)
	req.Amount = randAmount
	req.Method = "generate"
	req.Timestamp = int32(time.Now().Unix())
	auth := calculateAuth([]string{fmt.Sprintf("%d", req.Amount), req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), entities[0].CallbackSecret})
	req.AuthHash = auth
	resp := wsc.Request(req, nil)
	if !resp.Ok {
		t.Fatalf("request failed: %+v", resp)
	}

	if len(resp.Tokens) != randAmount {
		t.Fatalf("expected %d tokens, but got %d", randAmount, len(resp.Tokens))
	}

	// another entity cannot request
}

func TestStatus(t *testing.T) {
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
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/token", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// 0. Existing unused uuid should return available
	var req types.MetaRequest
	req.Method = "status"
	req.EntityID = fmt.Sprintf("%x", entities[0].ID)
	req.Token = membersUids[0].String()
	req.Timestamp = int32(time.Now().Unix())
	auth := calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
	req.AuthHash = auth
	resp := wsc.Request(req, nil)
	if !resp.Ok {
		t.Fatalf("request failed: %+v", resp)
	}

	if resp.TokenStatus != "available" {
		t.Fatalf("expected token \"available\", but got %s", resp.TokenStatus)
	}

	// 1. Existing used uuid should return registered
	s := new(ethereum.SignKeys)
	s.Generate()
	pubKey, _ := s.HexString()
	pub, err := hex.DecodeString(pubKey)
	if err != nil {
		t.Fatalf("unable to create pubKey: (%v)", err)
	}
	if err = api.DB.RegisterMember(entities[0].ID, pub, &membersUids[0]); err != nil {
		t.Fatalf("unable to register member: (%v)", err)
	}

	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	if !resp.Ok {
		t.Fatalf("request failed: %+v", resp)
	}

	if resp.TokenStatus != "registered" {
		t.Fatalf("expected token \"registered\", but got %s", resp.TokenStatus)
	}

	// 2. Non-Existing  uuid should return invalid
	req.Token = uuid.New().String()
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	if !resp.Ok {
		t.Fatalf("request failed: %+v", resp)
	}

	if resp.TokenStatus != "invalid" {
		t.Fatalf("expected token \"invalid\", but got %s", resp.TokenStatus)
	}

	// 3. Valid ID and Non-Existing  entity should return error invalid entity Id
	req.Token = membersUids[1].String()
	req.EntityID = "1234567"
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("expected  error message \"invalid entityId\", but request succeeded: %+v", resp)
	}

	if resp.Message != "invalid entityId" {
		t.Fatalf("expected  error message \"invalid entityId\", but got %s", resp.Message)
	}

	// 4. Invalid ID and Non-Existing  entity should return error invalid entity Id
	req.Token = membersUids[1].String()
	req.EntityID = "1234567"
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("expected  error message \"invalid entityId\", but request succeeded: %+v", resp)
	}

	if resp.Message != "invalid entityId" {
		t.Fatalf("expected  error message \"invalid entityId\", but got %s", resp.Message)
	}

	// 5. Valid ID and Empty  entity should return error invalid entity Id
	req.Token = membersUids[1].String()
	req.EntityID = ""
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("expected  error message \"invalid entityId\", but request succeeded: %+v", resp)
	}

	if resp.Message != "invalid entityId" {
		t.Fatalf("expected  error message \"invalid entityId\", but got %s", resp.Message)
	}

	// 6. Empty ID and Valid entity should return error invalid invalid token
	req.Token = ""
	req.EntityID = fmt.Sprintf("%x", entities[0].ID)
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("expected  error message \"invalid token\", but request succeeded: %+v", resp)
	}

	if resp.Message != "invalid token" {
		t.Fatalf("expected  error message \"invalid token\", but got %s", resp.Message)
	}

}

func TestRevoke(t *testing.T) {
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
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/token", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// 0. Existing uuid should return get revoked successfully
	var req types.MetaRequest
	req.Method = "revoke"
	req.EntityID = fmt.Sprintf("%x", entities[0].ID)
	req.Token = membersUids[0].String()
	req.Timestamp = int32(time.Now().Unix())
	auth := calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
	req.AuthHash = auth
	resp := wsc.Request(req, nil)
	if !resp.Ok {
		t.Fatalf("request failed: %+v", resp)
	}

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
	auth = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("succeed to revoke twice the same token %s : %v", req.Token, resp)
	}

	// 2. Non- Existing uuid should return error
	req.Token = uuid.New().String()
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("succeed to revoke a non-existing token: %v", resp)
	}

	// 3. Valid ID and Non-Existing  entity should return invalid entity Id
	req.Token = membersUids[1].String()
	req.EntityID = "1234567"
	req.Timestamp = int32(time.Now().Unix())
	auth = calculateAuth([]string{req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
	req.AuthHash = auth
	resp = wsc.Request(req, nil)
	if resp.Ok {
		t.Fatalf("succeed to revoke an existing token for a non-existing entity: %+v", resp)
	}

	if resp.Message != "invalid entityId" {
		t.Fatalf("expected  error message \"invalid entityId\", but got %s", resp.Message)
	}

}

func calculateAuth(fields []string) string {

	if len(fields) == 0 {
		return ""
	}
	toHash := ""
	for _, f := range fields {
		toHash += f
	}
	return hex.EncodeToString(ethereum.HashRaw([]byte(toHash)))
}
