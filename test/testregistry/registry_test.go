package testregistry

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

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
	api.Start(db, "/api")
	if err := api.DB.Ping(); err != nil {
		log.Printf("SKIPPING: could not connect to DB: %v", err)
		return
	}
	os.Exit(m.Run())
}

func TestRegister(t *testing.T) {
	// create entity
	_, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
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
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// register member without token
	var req types.MetaRequest
	// create register request
	req.Method = "register"
	req.EntityID = hex.EncodeToString(entities[0].ID)
	req.Member = &types.Member{
		MemberInfo: members[0].MemberInfo,
	}
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
				req2.EntityID = hex.EncodeToString(entities[0].ID)
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

func TestStatus(t *testing.T) {
	// create entity
	_, entities, err := testcommon.CreateEntities(1)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
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
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/registry", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// add user and member
	if err := api.DB.AddUser(&types.User{PubKey: members[0].PubKey}); err != nil {
		t.Fatalf("cannot add created user into database: %s", err)
	}
	if _, err := api.DB.AddMember(members[0].EntityID, members[0].PubKey, &members[0].MemberInfo); err != nil {
		t.Fatalf("cannot add created members into database: %s", err)
	}
	// check status added and linked member
	var req types.MetaRequest
	req.Method = "status"
	req.EntityID = hex.EncodeToString(entities[0].ID)
	resp := wsc.Request(req, membersSigners[0])
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
	req2.Method = "status"
	req2.EntityID = hex.EncodeToString(entities[0].ID)
	resp2 := wsc.Request(req2, membersSigners[1])
	if !resp2.Ok {
		t.Fatal()
	}
	if resp2.Status.Registered {
		t.Fatal("member should not be registered")
	}
}
