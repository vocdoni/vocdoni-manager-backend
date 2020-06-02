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
	AddMember(entityID []byte, pubKey string, info *types.MemberInfo) (*types.Member, error)
	Member(memberID uuid.UUID) (*types.Member, error)
	AddUser(user *types.User) error
	User(pubKey string) (*types.User, error)
	Census(censusID []byte) (*types.Census, error)
}
