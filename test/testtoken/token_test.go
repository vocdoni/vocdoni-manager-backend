package testregistry

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

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

func TestGenerateTokens(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/token", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	_, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
	entities[0].CallbackSecret = "test"
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into DB: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	randAmount := rand.Intn(100)
	req.EntityID = hex.EncodeToString(entities[0].ID)
	req.Amount = randAmount
	req.Method = "generate"
	req.Timestamp = int32(time.Now().Unix())
	auth := calculateAuth([]string{fmt.Sprintf("%d", req.Amount), req.EntityID, req.Method, fmt.Sprintf("%d", req.Timestamp), req.Token, entities[0].CallbackSecret})
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
