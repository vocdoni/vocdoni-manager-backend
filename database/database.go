package database

import (
	"database/sql"
)

type Database struct {
	db *sql.DB
}

func New(host string, port int, user, password, dbname string, sslmode string) (*Database, error) {
	return &Database{db: nil}, nil
}

func (d *Database) close() {
	d.db.Close()
}

// Entity returns the basic information of a registered entity
func (d *Database) Entity(entityID string) {
}
