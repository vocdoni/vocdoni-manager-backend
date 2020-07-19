package database

import (
	"github.com/google/uuid"
	migrate "github.com/rubenv/sql-migrate"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

type Database interface {
	Ping() error
	Close() error
	AddEntity(entityID []byte, info *types.EntityInfo) error
	Entity(entityID []byte) (*types.Entity, error)
	UpdateEntity(entityID []byte, info *types.EntityInfo) error
	EntityOrigins(entityID []byte) ([]types.Origin, error)
	EntityHas(entityID []byte, memberID *uuid.UUID) bool
	AddMember(entityID []byte, pubKey []byte, info *types.MemberInfo) (uuid.UUID, error)
	TempImportMembers(entityID []byte, info []types.MemberInfo) error
	ImportMembers(entityID []byte, info []types.MemberInfo) error
	AddMemberBulk(entityID []byte, members []types.Member) error
	Member(entityID []byte, memberID *uuid.UUID) (*types.Member, error)
	DeleteMember(entityID []byte, memberID *uuid.UUID) error
	MemberPubKey(entityID, pubKey []byte) (*types.Member, error)
	CountMembers(entityID []byte) (int, error)
	ListMembers(entityID []byte, filter *types.ListOptions) ([]types.Member, error)
	UpdateMember(entityID []byte, memberID *uuid.UUID, info *types.MemberInfo) error
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
	Census(entityID, censusID []byte) (*types.Census, error)
	AddCensus(entityID, censusID []byte, targetID *uuid.UUID, info *types.CensusInfo) error
	AddCensusWithMembers(entityID, censusID []byte, targetID *uuid.UUID, info *types.CensusInfo) (int64, error)
	CountCensus(entityID []byte) (int, error)
	ListCensus(entityID []byte) ([]types.Census, error)
	Migrate(dir migrate.MigrationDirection) (int, error)
	MigrateStatus() (int, int, string, error)
	MigrationUpSync() (int, error)
}
