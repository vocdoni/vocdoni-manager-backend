package database

import (
	"github.com/google/uuid"
	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/manager/types"
)

type Database interface {
	Ping() error
	Close() error
	AddEntity(entityID []byte, info *types.EntityInfo) error
	Entity(entityID []byte) (*types.Entity, error)
	DeleteEntity(entityID []byte) error
	EntitiesID() ([]string, error)
	AuthorizeEntity(entityID []byte) error
	UpdateEntity(entityID []byte, info *types.EntityInfo) (int, error)
	EntityOrigins(entityID []byte) ([]types.Origin, error)
	EntityHas(entityID []byte, memberID *uuid.UUID) bool
	AddMember(entityID []byte, pubKey []byte, info *types.MemberInfo) (uuid.UUID, error)
	ImportMembersWithPubKey(entityID []byte, info []types.MemberInfo) error
	ImportMembers(entityID []byte, info []types.MemberInfo) error
	AddMemberBulk(entityID []byte, members []types.Member) error
	Member(entityID []byte, memberID *uuid.UUID) (*types.Member, error)
	Members(entityID []byte, memberIDs []uuid.UUID) ([]types.Member, []uuid.UUID, error)
	MembersKeys(entityID []byte, memberKeys [][]byte) ([]types.Member, [][]byte, error)
	DeleteMember(entityID []byte, memberID *uuid.UUID) error
	DeleteMembers(entityID []byte, members []uuid.UUID) (int, []uuid.UUID, error)
	DeleteMembersByKeys(entityID []byte, memberKeys [][]byte) (int, [][]byte, error)
	MemberPubKey(entityID, pubKey []byte) (*types.Member, error)
	CountMembers(entityID []byte) (int, error)
	ListMembers(entityID []byte, filter *types.ListOptions) ([]types.Member, error)
	UpdateMember(entityID []byte, memberID *uuid.UUID, info *types.MemberInfo) (int, error)
	AddTag(entityID []byte, tagName string) (int32, error)
	DeleteTag(entityID []byte, tagID int32) error
	Tag(entityID []byte, tagID int32) (*types.Tag, error)
	TagByName(entityID []byte, tagName string) (*types.Tag, error)
	ListTags(entityID []byte) ([]types.Tag, error)
	AddTagToMembers(entityID []byte, members []uuid.UUID, tagID int32) (int, []uuid.UUID, error)
	RemoveTagFromMembers(entityID []byte, members []uuid.UUID, tagID int32) (int, []uuid.UUID, error)
	CreateMembersWithTokens(entityID []byte, tokens []uuid.UUID) error
	CreateNMembers(entityID []byte, n int) ([]uuid.UUID, error)
	RegisterMember(entityID, pubKey []byte, token *uuid.UUID) error
	MembersTokensEmails(entityID []byte) ([]types.Member, error)
	AddTarget(entityID []byte, target *types.Target) (uuid.UUID, error)
	Target(entityID []byte, targetID *uuid.UUID) (*types.Target, error)
	CountTargets(entityID []byte) (int, error)
	ListTargets(entityID []byte) ([]types.Target, error)
	TargetMembers(entityID []byte, targetID *uuid.UUID) ([]types.Member, error)
	AddUser(user *types.User) error
	User(pubKey []byte) (*types.User, error)
	DumpClaims(entityID []byte) ([][]byte, error)
	DumpCensusClaims(entityID []byte, censusID []byte) ([][]byte, error)
	ExpandCensusMembers(entityID, censusID []byte) ([]types.CensusMember, error)
	ListEphemeralMemberInfo(entityID, censusID []byte) ([]types.EphemeralMemberInfo, error)
	EphemeralMemberInfoByEmail(entityID, censusID []byte, email string) (*types.EphemeralMemberInfo, error)
	Census(entityID, censusID []byte) (*types.Census, error)
	UpdateCensus(entityID, censusID []byte, info *types.CensusInfo) (int, error)
	AddCensus(entityID, censusID []byte, targetID *uuid.UUID, info *types.CensusInfo) error
	AddCensusWithMembers(entityID, censusID []byte, targetID *uuid.UUID, info *types.CensusInfo) (int64, error)
	CountCensus(entityID []byte) (int, error)
	DeleteCensus(entityID []byte, censusID []byte) error
	ListCensus(entityID []byte, filter *types.ListOptions) ([]types.Census, error)
	Migrate(dir migrate.MigrationDirection) (int, error)
	MigrateStatus() (int, int, string, error)
	MigrationUpSync() (int, error)
}
