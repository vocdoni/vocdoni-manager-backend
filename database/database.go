package database

import (
	"github.com/google/uuid"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Database interface {
	Ping() error
	Close() error
	AddEntity(entityID []byte, info *types.EntityInfo) error
	Entity(entityID []byte) (*types.Entity, error)
	EntityOrigins(entityID []byte) ([]types.Origin, error)
	EntityHas(entityID []byte, memberID uuid.UUID) bool
	AddMember(entityID []byte, pubKey []byte, info *types.MemberInfo) (uuid.UUID, error)
	ImportMembers(entityID []byte, info []types.MemberInfo) error
	AddMemberBulk(entityID []byte, members []types.Member) error
	Member(entityID []byte, memberID uuid.UUID) (*types.Member, error)
	MemberPubKey(entityID, pubKey []byte) (*types.Member, error)
	CountMembers(entityID []byte) (int, error)
	ListMembers(entityID []byte, filter *types.ListOptions) ([]types.Member, error)
	UpdateMember(memberID uuid.UUID, pubKey []byte, info *types.MemberInfo) error
	CreateMembersWithTokens(entityID []byte, tokens []uuid.UUID) error
	MembersTokensEmails(entityID []byte) ([]types.Member, error)
	AddTarget(entityID []byte, target *types.Target) (uuid.UUID, error)
	Target(entityID []byte, targetID uuid.UUID) (*types.Target, error)
	ListTargets(entityID []byte) ([]types.Target, error)
	AddUser(user *types.User) error
	User(pubKey []byte) (*types.User, error)
	DumpClaims(entityID []byte) ([][]byte, error)
	Census(entityID, censusID []byte) (*types.Census, error)
	AddCensus(entityID, censusID []byte, targetID uuid.UUID, info *types.CensusInfo) error
	ListCensus(entityID []byte) ([]types.Census, error)
}
