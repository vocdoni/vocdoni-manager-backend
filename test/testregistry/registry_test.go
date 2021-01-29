package testregistry

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/dvote/crypto/ethereum"
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

func TestRegister(t *testing.T) {
	var err error
	// create entity
	_, entities := testcommon.CreateEntities(1)
	// add entity
	err = api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo)
	if err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}
	// create members
	membersSigners, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	// connect to endpoint
	wsc, err := testcommon.NewHTTPapiConnection(fmt.Sprintf("http://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// register member without token
	var req types.MetaRequest
	// create register request
	req.Method = "register"
	req.EntityID = entities[0].ID
	req.MemberInfo = &members[0].MemberInfo
	// make request
	resp := wsc.Request(req, membersSigners[0])
	// check register went successful
	if !resp.Ok {
		t.Fatal(err)
	}
	// check member created
	member, err := api.DB.MemberPubKey(members[0].EntityID, members[0].PubKey)
	if err != nil {
		t.Fatalf("cannot fetch member from the database: %s", err)
	}
	// check user added and member is linked with pubkey
	_, err = api.DB.User(member.PubKey)
	if err != nil {
		t.Fatalf("cannot fetch user from the Postgres DB (pgsql.go:User): %s", err)
	}
	// cannot register twice
	resp = wsc.Request(req, membersSigners[0])
	// check register failed
	if resp.Ok {
		t.Fatal("cannot add member twice")
	}

	// TBD: register with tokens

	/*
				// register member with token
				// create register request
				var req2 types.MetaRequest
				token1 := uuid.New()
				api.DB.CreateMembersWithTokens(entities[0].ID, []uuid.UUID{token1})
				req2.Token = strings.ReplaceAll(token1.String(), "-", "")
				req2.Method = "register"
				req2.EntityID = entities[0].ID
				req2.Member = &types.Member{
					MemberInfo: members[1].MemberInfo,
				}
				// make request
				resp2 := wsc.Request(req2, membersSigners[1])
				// check register went successful
				if !resp2.Ok {
					t.Error(err)
				}
				// check member created
		<<<<<<< HEAD
				member2, err := api.DB.Member(token1)
		=======
				member2, err := db.Member(entities[0].ID, token1)
		>>>>>>> Adds entityID as parameter to
				if err != nil {
					t.Errorf("Error getting member to the Postgres DB (pgsql.go:Member): %s", err)
				}
				mem1pubk, _ := membersSigners[1].HexString()
				if string(member2.PubKey) != mem1pubk {
					t.Errorf("member and user pubkey must match: {member: %s, user: %s}", member2.PubKey, mem1pubk)
				}

				// update member info
				// create register request
				var req3 types.MetaRequest
				req3.Method = "register"
				req3.Token = // token
				req3.EntityID = req.EntityID
				members[0].MemberInfo.Email = "emailchanged@vocdoni.io"
				req3.Member = &types.Member{
					MemberInfo: members[0].MemberInfo,
				}
				// make request
				resp3 := wsc.Request(req3, membersSigners[0])
				// check register went successful
				if !resp3.Ok {
					t.Error(err)
				}
				// check member updated
		<<<<<<< HEAD
				member3, err := api.DB.MemberPubKey(members[0].PubKey, members[0].EntityID)
		=======
				member3, err := db.MemberPubKey(members[0].EntityID, members[0].PubKey)
		>>>>>>> Adds entityID as parameter to
				if err != nil {
					t.Errorf("Error getting member to the Postgres DB (pgsql.go:addMember): %s", err)
				}
				if member3.MemberInfo.Email != members[0].MemberInfo.Email {
					t.Error("member email not updated")
				}

				// cannot reuse the same token
	*/
}

func TestValidateToken(t *testing.T) {
	// create entity
	var err error
	_, entities := testcommon.CreateEntities(2)
	// add entities
	err = api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo)
	if err != nil {
		t.Fatalf("cannot add entity into database: %s", err)
	}
	err = api.DB.AddEntity(entities[1].ID, &entities[1].EntityInfo)
	if err != nil {
		t.Fatalf("cannot add entity into database: %s", err)
	}
	// create tokens for 1st entity
	tokens, err := api.DB.CreateNMembers(entities[0].ID, 4)
	if err != nil {
		t.Fatalf("unable to create member using CreateNMembers:  (%+v)", err)
	}
	// create tokens for 2nd entity
	tokens2, err := api.DB.CreateNMembers(entities[1].ID, 1)
	if err != nil {
		t.Fatalf("unable to create member using CreateNMembers:  (%+v)", err)
	}
	// create signing keys
	membersSigners, _, err := testcommon.CreateMembers(entities[1].ID, 3)
	if err != nil {
		t.Fatalf("cannot create member signer: %v", err)
	}
	tagID, err := api.DB.AddTag(entities[0].ID, "PendingValidation")
	if err != nil {
		t.Fatalf("cannot create PendingValidation tag: %v", err)
	}
	if _, _, err := api.DB.AddTagToMembers(entities[0].ID, []uuid.UUID{tokens[0]}, tagID); err != nil {
		t.Fatalf("cannot add PendingValidation tag to member: %v", err)
	}

	// connect to endpoint
	wsc, err := testcommon.NewHTTPapiConnection(fmt.Sprintf("http://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// register member without token
	var req types.MetaRequest
	// create register request
	req.Method = "validateToken"
	req.EntityID = entities[0].ID
	req.Token = tokens[0].String()
	// make request
	resp := wsc.Request(req, membersSigners[0])
	// check register went successful
	if !resp.Ok {
		t.Fatal(err)
	}
	// 1. check member created
	member, err := api.DB.Member(entities[0].ID, &tokens[0])
	if err != nil {
		t.Fatalf("cannot fetch validated member from the database: %s", err)
	}
	if len(member.Tags) > 0 {
		t.Fatal("PendingValidation tag was not removed from member")
	}
	// 2. check user added and member is linked with pubkey
	_, err = api.DB.User(member.PubKey)
	if err != nil {
		t.Fatalf("cannot fetch corresponding validated user from the Postgres DB (pgsql.go:User): %s", err)
	}
	// 3. cannot validate same token twice
	resp = wsc.Request(req, membersSigners[0])
	// check register failed
	if resp.Ok || resp.Message != "duplicate user already registered" {
		t.Fatal("validated same token  with same pubKey")
	}

	// 4. cannot validate same token twice
	resp = wsc.Request(req, membersSigners[1])
	// check register failed
	if resp.Ok || resp.Message != "invalid token" {
		t.Fatal("validated same token twice with different pubKeys ")
	}

	// 4. check cannot validate random token (with new signer)
	req.Token = uuid.New().String()
	resp = wsc.Request(req, membersSigners[1])
	// check register failed
	if resp.Ok {
		t.Fatal("validated random token")
	}

	// 5. check cannot validate correct token in non-existing entity (with new signer)
	req.Token = tokens[1].String()
	req.EntityID = entities[1].ID
	resp = wsc.Request(req, membersSigners[1])
	// check register failed
	if resp.Ok {
		t.Fatal(" validated correct token in non-existing entity")
	}

	// 6. check cannot validate correct token in existing non-corresponding entity (with new signer)
	// add entity
	req.Token = tokens[1].String()
	req.EntityID = entities[1].ID
	resp = wsc.Request(req, membersSigners[1])
	// check register failed
	if resp.Ok {
		t.Fatal("validated correct token in existing non-corresponding entity")
	}

	// 7. check cannot reuse the same pubKey to validate a new token
	req.EntityID = entities[0].ID
	req.Token = tokens[3].String()
	resp = wsc.Request(req, membersSigners[0])
	// check register failed
	if resp.Ok {
		t.Fatal("reused the same pubKey to validate a new token")
	}

	// 8. Test callback fails with wrong event type
	port := "12000"
	secret := "awsedrft"
	ts := "1000"
	event := "register"
	urlParameters := "?authHash={AUTH}&event={EVENT}&timestamp={TIMESTAMP}&token={TOKEN}"
	h := ethereum.HashRaw([]byte(event + ts + tokens[2].String() + secret))

	updatedInfo := &types.EntityInfo{
		CallbackURL:    "http://127.0.0.1:" + port + urlParameters,
		CallbackSecret: secret,
	}

	err = api.DB.UpdateEntity(entities[0].ID, updatedInfo)
	if err != nil {
		t.Fatalf("cannot fetch validated member from the database: %s", err)
	}

	params := map[string]string{
		"authHash":  fmt.Sprintf("%x", h),
		"event":     event,
		"token":     tokens[2].String(),
		"timestamp": ts,
	}

	req.Token = tokens[2].String()
	req.EntityID = entities[0].ID
	s := testcommon.TestCallbackServer(t, port, params)
	resp = wsc.Request(req, membersSigners[2])
	s.Close()
	if !resp.Ok {
		t.Fatal("cannot validate member using also callback")
	}

	// check member created
	member, err = api.DB.Member(entities[0].ID, &tokens[2])
	if err != nil {
		t.Fatalf("cannot fetch validated member from the database: %s", err)
	}
	// check user added and member is linked with pubkey
	_, err = api.DB.User(member.PubKey)
	if err != nil {
		t.Fatalf("cannot fetch corresponding validated user from the Postgres DB (pgsql.go:User): %s", err)
	}

	// Example of Running and expecting to fail for the callback
	// without making the test fail
	//
	// params["event"] = "random"
	// result := t.Run("fail", func(t *testing.T) {
	// 	s := testcommon.TestCallbackServer(t, port, params)
	// 	resp = wsc.Request(req, membersSigners[2])
	// 	s.Close()
	// })
	// if result != fail {
	// 	t.Fatalf("Callback  accetps illegal event (\"random\"")
	// }

	// 11. check can reuse the same pubKey to validate a new token for another entity
	req.EntityID = entities[1].ID
	req.Token = tokens2[0].String()
	resp = wsc.Request(req, membersSigners[0])
	// check register failed
	if !resp.Ok {
		t.Fatal("cannot reuse the same pubKey to validate a new token for a new entity")
	}

}

func TestStatus(t *testing.T) {
	var err error
	// create entity
	_, entities := testcommon.CreateEntities(1)
	// add entity
	err = api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo)
	if err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// create members
	membersSigners, members, err := testcommon.CreateMembers(entities[0].ID, 2)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	// connect to endpoint
	wsc, err := testcommon.NewHTTPapiConnection(fmt.Sprintf("http://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}

	// check status added and linked member
	var req types.MetaRequest
	req.Method = "registrationStatus"
	req.EntityID = entities[0].ID
	resp := wsc.Request(req, membersSigners[0])
	if !resp.Ok {
		t.Fatal()
	}
	if resp.Status.Registered {
		t.Fatal("member should not be registered")
	}

	// add user and member
	if err := api.DB.AddUser(&types.User{PubKey: members[0].PubKey}); err != nil {
		t.Fatalf("cannot add created user into database: %s", err)
	}
	if _, err := api.DB.AddMember(members[0].EntityID, members[0].PubKey, &members[0].MemberInfo); err != nil {
		t.Fatalf("cannot add created members into database: %s", err)
	}
	// check status added and linked member
	// var req types.MetaRequest
	req.Method = "registrationStatus"
	req.EntityID = entities[0].ID
	resp = wsc.Request(req, membersSigners[0])
	if !resp.Ok {
		t.Fatal()
	}
	if !resp.Status.Registered {
		t.Fatal("member should be registered")
	}
	// check status non registered member
	if err := api.DB.AddUser(&types.User{PubKey: members[1].PubKey}); err != nil {
		t.Fatalf("cannot add user into database: %s", err)
	}
	// check user not registered
	var req2 types.MetaRequest
	req2.Method = "registrationStatus"
	req2.EntityID = entities[0].ID
	resp2 := wsc.Request(req2, membersSigners[1])
	if !resp2.Ok {
		t.Fatal()
	}
	if resp2.Status.Registered {
		t.Fatal("member should not be registered")
	}
}
