package testdb

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/dvote/crypto/snarks"
	"go.vocdoni.io/manager/types"
)

var Signers = []struct {
	Pub  string
	Priv string
}{
	// User() no rows
	{"03f9d1e41906436bf2e8aab319383dea6a4c06426266955293fd92b41f6346256f", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca375"},
	// user no rows failed AddUser()
	{"03733ca0d2462ef3cd4dbd331d5ec27a63eeb13afbaf03f236847479c3e8d7fd94", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca355"},
	// failed User()
	{"0399d0ad8447520e66df7db954b0936f4b141a01ba6213dda88c9df7293b66262e", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca357"},
	// MemberPubKey() no rows
	{"026163a9bc3425426bbb7f0fde6c9bb4504493415a34b99a84162fe01640a784a3", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca372"},
}

type Database struct {
}

func New() (*Database, error) {
	return &Database{}, nil
}

func (d *Database) Ping() error {
	return nil
}

func (d *Database) Close() error {
	return nil
}

func (d *Database) Entity(entityID []byte) (*types.Entity, error) {
	if fmt.Sprintf("%x", entityID) == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return nil, fmt.Errorf("error adding entity with id: %x", entityID)
	}
	if fmt.Sprintf("%x", entityID) == "f6da3e4864d566faf82163a407e84a9001592678" {
		return nil, fmt.Errorf("cannot fetch entity with ID: %x", entityID)
	}

	var entity types.Entity
	entity.ID = entityID
	managerAddresses, err := hex.DecodeString("02ed03e6408e34af72a0e062a50cd9e77997c6c0eded5835b7367bb5695e844bf4")
	if err != nil {
		return nil, fmt.Errorf("error decoding manager address: %s", err)
	}
	entity.CensusManagersAddresses = [][]byte{managerAddresses}
	entity.Name = "test entity"
	entity.Email = "entity@entity.org"

	failEidID := hex.EncodeToString(entityID)
	if failEidID == "ca526af2aaa0f3e9bb68ab80de4392590f7b153a" {
		entity.ID = []byte{1}
	}

	return &entity, nil
}

func (d *Database) EntitiesID() ([]string, error) {
	return nil, nil
}

func (d *Database) EntityHas(entityID []byte, memberID *uuid.UUID) bool {
	return true
}

func (d *Database) EntityOrigins(entityID []byte) ([]types.Origin, error) {
	return nil, nil
}

func (d *Database) AddEntity(entityID []byte, info *types.EntityInfo) error {
	if fmt.Sprintf("%x", entityID) == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return fmt.Errorf("error adding entity with id: %x", entityID)
	}
	return nil
}

func (d *Database) DeleteEntity(entityID []byte) error {
	if fmt.Sprintf("%x", entityID) == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return fmt.Errorf("error deleting entity with id: %x", entityID)
	}
	return nil
}

func (d *Database) AuthorizeEntity(entityID []byte) error {
	if fmt.Sprintf("%x", entityID) == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return fmt.Errorf("error adding entity with id: %x", entityID)
	}
	return nil
}

func (d *Database) UpdateEntity(entityID []byte, info *types.EntityInfo) (int, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return 0, fmt.Errorf("error updating entity with id: %s", failEid)
	}
	return 1, nil
}

func (d *Database) Member(entityID []byte, memberID *uuid.UUID) (*types.Member, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return nil, sql.ErrNoRows
	}
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return nil, fmt.Errorf("cannot get member")
	}
	var member types.Member
	if memberID != nil {
		member.ID = *memberID
	}
	member.EntityID = entityID
	member.Email = "hello@vocdoni.io"
	member.FirstName = "Julian"
	member.LastName = "Assange"
	member.Phone = "+441827738192"
	member.PubKey = []byte("020be846bab70b4eff964d74178187832b3c7866f8509de340b6fccc53032834c6")
	member.DateOfBirth = time.Time{}
	member.StreetAddress = "Yolo St. 550"
	return &member, nil
}

func (d *Database) Members(entityID []byte, memberIDs []uuid.UUID) ([]types.Member, []uuid.UUID, error) {
	return nil, nil, nil
}

func (d *Database) MembersKeys(entityID []byte, memberKeys [][]byte) ([]types.Member, [][]byte, error) {
	return nil, nil, nil
}

func (d *Database) MemberPubKey(entityID, pubKey []byte) (*types.Member, error) {
	failPub := hex.EncodeToString(pubKey)
	if failPub == Signers[3].Pub {
		return nil, sql.ErrNoRows
	}
	var member types.Member
	member.ID = uuid.New()
	member.EntityID = entityID
	member.Email = "hello@vocdoni.io"
	member.FirstName = "Julian"
	member.LastName = "Assange"
	member.Phone = "+441827738192"
	member.PubKey = []byte("020be846bab70b4eff964d74178187832b3c7866f8509de340b6fccc53032834c6")
	member.DateOfBirth = time.Time{}
	member.StreetAddress = "Yolo St. 550"
	return &member, nil
}

func (d *Database) CountMembers(entityID []byte) (int, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return 0, fmt.Errorf("error counting members of entity: %s", failEid)
	}
	return 0, nil
}

func (d *Database) ListMembers(entityID []byte, filter *types.ListOptions) ([]types.Member, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return nil, sql.ErrNoRows
	}
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return nil, fmt.Errorf("cannot list members")
	}
	return nil, nil
}

func (d *Database) Census(entityID, censusID []byte) (*types.Census, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return nil, fmt.Errorf("error getting census of entity: %s", failEid)
	}
	var census types.Census
	census.ID = []byte("0x0")
	return &census, nil
}

func (d *Database) CountCensus(entityID []byte) (int, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return 0, fmt.Errorf("error counting census of entity: %s", failEid)
	}
	return 0, nil
}

func (d *Database) ListCensus(entityID []byte, filter *types.ListOptions) ([]types.Census, error) {
	if fmt.Sprintf("%x", entityID) == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return nil, sql.ErrNoRows
	}
	if fmt.Sprintf("%x", entityID) == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return nil, fmt.Errorf("cannot list census from entity: %x", entityID)
	}
	return nil, nil
}

func (d *Database) AddCensus(entityID, censusID []byte, targetID *uuid.UUID, info *types.CensusInfo) error {
	failEid := hex.EncodeToString(entityID)
	if failEid == "c87363d9919daef530bf19e907df7f2d8920be75" {
		return fmt.Errorf("cannot add census to entity: %s", failEid)
	}
	return nil
}

func (d *Database) DeleteCensus(entityID []byte, censusID []byte) error {
	failEid := hex.EncodeToString(entityID)
	if failEid == "c87363d9919daef530bf19e907df7f2d8920be75" {
		return fmt.Errorf("cannot add census to entity: %s", failEid)
	}
	return nil
}

func (d *Database) AddCensusWithMembers(entityID, censusID []byte, targetID *uuid.UUID, info *types.CensusInfo) (int64, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "c87363d9919daef530bf19e907df7f2d8920be75" {
		return 0, fmt.Errorf("cannot add census members to entity: %s", failEid)
	}
	return 0, nil
}

func (d *Database) AddMember(entityID []byte, pubKey []byte, info *types.MemberInfo) (uuid.UUID, error) {
	// return &types.Member{MemberInfo: *info, ID: uuid.New(), EntityID: entityID, PubKey: pubKey}, nil
	if info.Email == "fail@fail.fail" {
		return uuid.Nil, errors.New("cannot add member, invalid mock email")
	}
	failEid := hex.EncodeToString(entityID)
	if failEid == "f6da3e4864d566faf82163a407e84a9001592678" {
		return uuid.Nil, fmt.Errorf("cannot fetch entity with ID: %s", failEid)
	}
	failPub := hex.EncodeToString(pubKey)
	if failPub == Signers[1].Pub {
		return uuid.Nil, fmt.Errorf("error adding user with pubKey: %s", failPub)
	}
	if failPub == Signers[2].Pub {
		return uuid.Nil, fmt.Errorf("cannot query for user")
	}
	return uuid.Nil, nil
}

func (d *Database) DeleteMember(entityID []byte, memberID *uuid.UUID) error {
	failEid := hex.EncodeToString(entityID)
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return fmt.Errorf("error deleting member of entity: %s", failEid)
	}
	return nil
}

func (d *Database) DeleteMembers(entityID []byte, members []uuid.UUID) (int, []uuid.UUID, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return 0, nil, fmt.Errorf("error deleting members of entity: %s", failEid)
	}
	return len(members), []uuid.UUID{}, nil
}

func (d *Database) DeleteMembersByKeys(entityID []byte, memberKeys [][]byte) (int, [][]byte, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return 0, nil, fmt.Errorf("error deleting members of entity: %s", failEid)
	}
	return len(memberKeys), [][]byte{}, nil
}

func (d *Database) ImportMembersWithPubKey(entityID []byte, info []types.MemberInfo) error {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return fmt.Errorf("error importing members of entity: %s", failEid)
	}
	return nil
}

func (d *Database) ImportMembers(entityID []byte, info []types.MemberInfo) error {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return fmt.Errorf("error importing members of entity: %s", failEid)
	}
	return nil
}

func (d *Database) AddMemberBulk(entityID []byte, members []types.Member) error {
	return nil
}

func (d *Database) UpdateMember(entityID []byte, memberID *uuid.UUID, info *types.MemberInfo) (int, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return 0, fmt.Errorf("error updating member of entity: %s", failEid)
	}
	return 1, nil
}

func (d *Database) AddTag(entityID []byte, tagName string) (int32, error) {
	return 1, nil
}
func (d *Database) ListTags(entityID []byte) ([]types.Tag, error) {
	return nil, nil
}
func (d *Database) DeleteTag(entityID []byte, tagID int32) error {
	return nil
}
func (d *Database) Tag(entityID []byte, tagID int32) (*types.Tag, error) {
	return nil, nil
}

func (d *Database) TagByName(entityID []byte, tagName string) (*types.Tag, error) {
	return nil, nil
}

func (d *Database) AddTagToMembers(entityID []byte, members []uuid.UUID, tagID int32) (int, []uuid.UUID, error) {
	return len(members), []uuid.UUID{}, nil
}

func (d *Database) RemoveTagFromMembers(entityID []byte, members []uuid.UUID, tagID int32) (int, []uuid.UUID, error) {
	return len(members), []uuid.UUID{}, nil
}

func (d *Database) CreateMembersWithTokens(entityID []byte, tokens []uuid.UUID) error {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return fmt.Errorf("cannot create members")
	}
	return nil
}

func (d *Database) CreateNMembers(entityID []byte, n int) ([]uuid.UUID, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return nil, fmt.Errorf("cannot create members")
	}
	return nil, nil
}

func (d *Database) RegisterMember(entityID, pubKey []byte, token *uuid.UUID) error {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return fmt.Errorf("cannot create members")
	}
	return nil
}

func (d *Database) MembersTokensEmails(entityID []byte) ([]types.Member, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return nil, sql.ErrNoRows
	}
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return nil, fmt.Errorf("error exporting members of entity: %s", failEid)
	}
	return nil, nil
}
func (d *Database) DumpClaims(entityID []byte) ([][]byte, error) {
	return nil, nil
}

func (d *Database) DumpCensusClaims(entityID []byte, censusID []byte) ([][]byte, error) {
	return nil, nil
}

func (d *Database) ExpandCensusMembers(entityID, censusID []byte) ([]types.CensusMember, error) {
	return nil, nil
}

func (d *Database) ListEphemeralMemberInfo(entityID, censusID []byte) ([]types.EphemeralMemberInfo, error) {
	return nil, nil
}

func (d *Database) EphemeralMemberInfoByEmail(entityID, censusID []byte, email string) (*types.EphemeralMemberInfo, error) {
	return nil, nil
}

func (d *Database) UpdateCensus(entityID, censusID []byte, info *types.CensusInfo) (int, error) {
	return 1, nil
}

func (d *Database) AddTarget(entityID []byte, target *types.Target) (uuid.UUID, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return uuid.Nil, fmt.Errorf("cannot add target: %+v", target)
	}
	return uuid.Nil, nil
}
func (d *Database) Target(entityID []byte, targetID *uuid.UUID) (*types.Target, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return nil, fmt.Errorf("error getting target from entity: %s", failEid)
	}
	if failEid == "6d3e07d7d1dd84469cc3adf49fa83daf2678b4c9" {
		t1 := types.Target{}
		return &t1, fmt.Errorf("error listing targets of entity: %s", failEid)
	}
	return nil, nil
}
func (d *Database) ListTargets(entityID []byte) ([]types.Target, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "c87363d9919daef530bf19e907df7f2d8920be75" {
		return nil, sql.ErrNoRows
	}
	if failEid == "6d3e07d7d1dd84469cc3adf49fa83daf2678b4c9" {
		t1 := make([]types.Target, 1)
		t1[0] = types.Target{}
		return t1, fmt.Errorf("error listing targets of entity: %s", failEid)
	}
	return nil, nil
}
func (d *Database) CountTargets(entityID []byte) (int, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return 0, fmt.Errorf("error counting targets of entity: %s", failEid)
	}
	return 0, nil
}
func (d *Database) TargetMembers(entityID []byte, targetID *uuid.UUID) ([]types.Member, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "5fa506aa68191bcc657795e57f080472e712c27d" {
		return nil, fmt.Errorf("error targeting members of entity: %s", failEid)
	}
	return nil, nil
}

func (d *Database) AddUser(user *types.User) error {
	failPub := hex.EncodeToString(user.PubKey)
	if failPub == Signers[1].Pub {
		return fmt.Errorf("error adding user with pubKey: %s", failPub)
	}
	return nil
}

func (d *Database) User(pubKey []byte) (*types.User, error) {
	failPub := hex.EncodeToString(pubKey)
	if failPub == Signers[0].Pub ||
		failPub == Signers[1].Pub {
		return nil, sql.ErrNoRows
	}
	if failPub == Signers[2].Pub {
		return nil, fmt.Errorf("cannot query for user")
	}
	return &types.User{
		PubKey:         pubKey,
		DigestedPubKey: snarks.Poseidon.Hash(pubKey),
	}, nil
}

func (d *Database) Migrate(dir migrate.MigrationDirection) (int, error) {
	return 0, nil
}

func (d *Database) MigrateStatus() (int, int, string, error) {
	return 0, 0, "", nil
}

func (d *Database) MigrationUpSync() (int, error) {
	return 0, nil
}
