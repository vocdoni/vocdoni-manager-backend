package database

import (
	"github.com/google/uuid"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Database interface {
	Close() error
	AddEntity(entityID string, info *types.EntityInfo) error
	Entity(entityID string) (*types.Entity, error)
	EntityHas(entityID string, memberID uuid.UUID) bool
	AddMember(entityID string, pubKey string, info *types.MemberInfo) (*types.Member, error)
	Member(memberID uuid.UUID) (*types.Member, error)
	Census(censusID string) (*types.Census, error)
}
