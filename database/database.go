package database

import (
	"github.com/google/uuid"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Database interface {
	Close() error
	Entity(entityID string) (*types.Entity, error)
	EntityHas(entityID string, memberID uuid.UUID) bool
	Member(memberID uuid.UUID) (*types.Member, error)
	Census(censusID string) (*types.Census, error)
}
