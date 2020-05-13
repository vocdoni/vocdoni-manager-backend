package database

import (
	"database/sql"
	"fmt"
)

type Database struct {
	db *sql.DB
}

func New(host string, port int, user, password, dbname string, sslmode string) (*Database, error) {
	psql := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", psql)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the databae host %s: (%s)", host, err)
	}
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("database host %s does not reply: (%s)", host, err)
	}
	return &Database{db: db}, nil
}

func (d *Database) close() {
	d.db.Close()
}

