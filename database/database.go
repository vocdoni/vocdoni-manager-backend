package database

import (
	"github.com/google/uuid"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Database interface {
	Close() error
	CreateEntity(entityID string, info *types.EntityInfo) (*types.Entity, error)
	Entity(entityID string) (*types.Entity, error)
	EntityHas(entityID string, memberID uuid.UUID) bool
	CreateMember(entityID, publicKey string, member *types.MemberInfo) (*types.Member, error)
	Member(memberID uuid.UUID) (*types.Member, error)
	Census(censusID string) (*types.Census, error)
}
