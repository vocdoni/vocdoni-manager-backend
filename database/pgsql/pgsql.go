package pgsql

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/stdlib"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Database struct {
	db *sqlx.DB
}

func New(dbc *config.DB) (*Database, error) {

	db, err := sqlx.Open("pgx", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s ",
		dbc.Host, dbc.Port, dbc.User, dbc.Password, dbc.Dbname, dbc.Sslmode))
	if err != nil {
		log.Fatal(err)
	}
	return &Database{db: db}, err
}

func (d *Database) Close() error {
	defer d.db.Close()
	return nil
	// return d.db.Close()
}

func (d *Database) AddEntity(entity *types.Entity) error {
	insert := `INSERT INTO entities
					(id, address, email, name, census_managers_addresses)
					VALUES (:id, :address, :email, :name, :pg_census_managers_addresses)`
	pgEntity := ToPGEntity(entity)
	_, err := d.db.NamedExec(insert, pgEntity)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (d *Database) Entity(entityID string) (*types.Entity, error) {
	var pgEntity PGEntity
	selectQuery := `SELECT id, address, email, name, census_managers_addresses as "pg_census_managers_addresses"  FROM entities WHERE id=$1`
	row := d.db.QueryRowx(selectQuery, entityID)
	err := row.StructScan(&pgEntity)
	entity := ToEntity(&pgEntity)
	js, _ := json.Marshal(entity)
	log.Debugf("%s", string(js))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return entity, nil
}

func (d *Database) EntityHas(entityID string, memberID uuid.UUID) bool {
	return true
}

func (d *Database) CreateEntity(entityID string, info *types.EntityInfo) (*types.Entity, error) {
	return &types.Entity{EntityInfo: *info}, nil
}

func (d *Database) CreateMember(entityID, pubKey string, info *types.MemberInfo) (*types.Member, error) {
	// pgmember := ToPGMember(member)
	// insert := `INSERT INTO members
	//  				(entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields)
	// 				 VALUES (:entity_id, :public_key, :street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified, :pg_custom_fields)`
	// _, err := d.db.NamedExec(insert, pgmember)
	// if err != nil {
	// 	log.Error(err)
	// 	return nil, err
	// }
	return &types.Member{MemberInfo: *info}, nil
}

func (d *Database) Member(memberID uuid.UUID) (*types.Member, error) {
	var pgMember PGMember
	selectQuery := `SELECT
	 				entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields as "pg_custom_fields"
					FROM members WHERE id =$1`
	row := d.db.QueryRowx(selectQuery, memberID)
	err := row.StructScan(&pgMember)
	member := ToMember(&pgMember)
	js, _ := json.Marshal(member)
	log.Debugf("%s", string(js))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return member, nil
}

func (d *Database) Census(censusID string) (*types.Census, error) {
	var census types.Census
	census.ID = uuid.New().String()
	return &census, nil
}

// func (p *types.MembersCustomFields) Value() (driver.Value, error) {
// 	j, err := json.Marshal(p)
// 	return j, err
// }

// type StringArray []string

// func (p []string) Value() (driver.Value, error) {
// 	j, err := json.Marshal(p)
// 	return j, err
// }
