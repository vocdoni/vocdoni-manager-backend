package pgsql

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Database struct {
	db *sqlx.DB
}

func New(host string, port int, user, password, dbname string, sslmode string) (*Database, error) {
	db, err := sqlx.Connect("postgres", fmt.Sprintf("dbname=%s user=%s", dbname, user))
	return &Database{db: db}, err
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) Entity(entityID string) (*types.Entity, error) {
	var entity types.Entity
	entity.ID = "0x12345123451234"
	entity.Address = "0x123847192347"
	entity.Name = "test entity"
	return &entity, nil
}

func (d *Database) EntityHas(entityID string, memberID uuid.UUID) bool {
	return true
}

func (d *Database) Member(memberID uuid.UUID) (*types.Member, error) {
	var member types.Member
	member.ID = uuid.New()
	return &member, nil
}

func (d *Database) Census(censusID string) (*types.Census, error) {
	var census types.Census
	census.ID = uuid.New().String()
	return &census, nil
}
