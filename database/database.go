package database

import (
	"github.com/google/uuid"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Database interface {
	Close() error
	CreateEntity(entityID string, address string, email string, name string, censusManagersAddresses []string) (*types.Entity, error)
	Entity(entityID string) (*types.Entity, error)
	EntityHas(entityID string, memberID uuid.UUID) bool
	CreateMember(entityID string) (*types.Member, error)
	Member(memberID uuid.UUID) (*types.Member, error)
	Census(censusID string) (*types.Census, error)
}
