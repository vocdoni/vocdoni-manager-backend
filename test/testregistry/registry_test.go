package testregistry

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/util"
	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
	"gitlab.com/vocdoni/vocdoni-manager-backend/test/testcommon"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

var api testcommon.TestAPI

func TestMain(t *testing.M) {
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

func TestRegister(t *testing.T) {
	db := api.DB
	var req types.MetaRequest
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Error(err)
	}

	// create entity
	entitySigner := new(ethereum.SignKeys)
	entitySigner.Generate()
	entityAddress := entitySigner.EthAddrString()
	eid, err := hex.DecodeString(util.TrimHex(entityAddress))
	if err != nil {
		t.Errorf("error decoding entity address: %s", err)
	}

	entityID := ethereum.HashRaw(eid)
	info := &types.EntityInfo{
		Address: eid,
		// Email:                   "entity@entity.org",
		Name:                    "test entity",
		CensusManagersAddresses: [][]byte{{1, 2, 3}},
		Origins:                 []types.Origin{types.Token.Origin()},
	}

	err = db.AddEntity(entityID, info)
	if err != nil {
		t.Errorf("Error adding entity to the Postgres DB (pgsql.go:addEntity): %s", err)
	}

	// without token non existing user

	s := new(ethereum.SignKeys)
	// generate signing keys
	s.Generate()
	// create register request
	req.Method = "register"
	req.EntityID = hex.EncodeToString(entityID)
	req.PubKey, _ = s.HexString()
	mInfo := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo}
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if !resp.Ok {
		t.Error(err)
	}

	// with token existing user

	// Create pubkey and Add membmer to the db
	memberSigner := new(ethereum.SignKeys)
	memberSigner.Generate()
	memberSignerPubKey, _ := memberSigner.HexString()
	memberInfo := &types.MemberInfo{}
	memberInfo.DateOfBirth.Round(time.Microsecond).UTC()
	memberInfo.Verified.Round(time.Microsecond)
	user := &types.User{PubKey: []byte(memberSignerPubKey)}
	err = db.AddUser(user)
	if err != nil {
		t.Errorf("Error adding user to the Postgres DB (pgsql.go:addUser) %s", err)
	}
	err = db.AddMember(entityID, user.PubKey, memberInfo)
	if err != nil {
		t.Errorf("Error adding member to the Postgres DB (pgsql.go:addMember): %s", err)
	}
	// get created member
	mem, err := db.MemberPubKey(user.PubKey, entityID)
	mem.MemberInfo.Email = "info2@vocdoni.io"
	// create request
	var req2 types.MetaRequest
	req2.Token = strings.ReplaceAll(mem.ID.String(), "-", "")
	req2.Method = "register"
	req2.EntityID = hex.EncodeToString(entityID)
	req2.Member = mem
	// make request
	resp2 := wsc.Request(req2, memberSigner)
	// check register went successful
	if !resp2.Ok {
		t.Error(err)
	}

}

func TestStatus(t *testing.T) {
	db := api.DB

	var req types.MetaRequest
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Error(err)
	}

	// create entity
	entitySigner := new(ethereum.SignKeys)
	entitySigner.Generate()
	entityAddress := entitySigner.EthAddrString()
	eid, err := hex.DecodeString(util.TrimHex(entityAddress))
	if err != nil {
		t.Errorf("error decoding entity address: %s", err)
	}

	entityID := ethereum.HashRaw(eid)
	info := &types.EntityInfo{
		Address: eid,
		// Email:                   "entity@entity.org",
		Name:                    "test entity",
		CensusManagersAddresses: [][]byte{{1, 2, 3}},
		Origins:                 []types.Origin{types.Token.Origin()},
	}

	err = db.AddEntity(entityID, info)
	if err != nil {
		t.Errorf("Error adding entity to the Postgres DB (pgsql.go:addEntity): %s", err)
	}

	// create member

	s := new(ethereum.SignKeys)
	// generate signing keys
	s.Generate()
	// create register request
	req.Method = "register"
	req.EntityID = hex.EncodeToString(entityID)
	req.PubKey, _ = s.HexString()
	mInfo := types.MemberInfo{
		Email: "info@vocdoni.io",
	}
	req.Member = &types.Member{ID: uuid.New(), MemberInfo: mInfo}
	// make request
	resp := wsc.Request(req, s)
	// check register went successful
	if !resp.Ok {
		t.Error(err)
	}

	var req2 types.MetaRequest
	req2.Method = "status"
	req2.EntityID = hex.EncodeToString(entityID)
	resp2 := wsc.Request(req2, s)
	if !resp2.Ok {
		t.Error(err)
	}
	if !resp2.Status.Registered {
		t.Error(err)
	}
	if resp2.Status.NeedsUpdate {
		t.Error(err)
	}
}
