package database

import (
	"github.com/google/uuid"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Database interface {
	Close() error
	AddEntity(entityID []byte, info *types.EntityInfo) error
	Entity(entityID []byte) (*types.Entity, error)
	EntityOrigins(entityID []byte) ([]types.Origin, error)
	EntityHas(entityID []byte, memberID uuid.UUID) bool
	AddMember(entityID []byte, pubKey []byte, info *types.MemberInfo) error
	AddMemberBulk(entityID []byte, info []types.MemberInfo) error
	Member(memberID uuid.UUID) (*types.Member, error)
	MemberPubKey(pubKey, entityID []byte) (*types.Member, error)
	ListMembers(entityID []byte, filter *types.ListOptions) ([]types.Member, error)
	UpdateMember(memberID uuid.UUID, pubKey []byte, info *types.MemberInfo) error
	CreateMembersWithTokens(entityID []byte, tokens []uuid.UUID) error
	MembersTokensEmails(entityID []byte) ([]types.Member, error)
	AddUser(user *types.User) error
	User(pubKey []byte) (*types.User, error)
	Census(censusID []byte) (*types.Census, error)
}
