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
	MembersFiltered(entityID []byte, info *types.MemberInfo, filter *types.Filter) ([]*types.Member, error)
	UpdateMember(memberID uuid.UUID, pubKey []byte, info *types.MemberInfo) error
	AddUser(user *types.User) error
	User(pubKey []byte) (*types.User, error)
	Census(censusID []byte) (*types.Census, error)
}
