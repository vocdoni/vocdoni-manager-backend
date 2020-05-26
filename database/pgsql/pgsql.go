package pgsql

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	_ "github.com/jackc/pgx/stdlib"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Database struct {
	db *sqlx.DB
}

func New(host string, port int, user, password, dbname string, sslmode string) (*Database, error) {

	db, err := sqlx.Open("pgx", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s ", host, port, user, password, dbname, sslmode))
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

func (d *Database) CreateEntity(entityID string, address string, email string, name string, censusManagersAddresses []string) (*types.Entity, error) {
	insert := `INSERT INTO entities
					(id, address, email, name, census_managers_addresses)
					VALUES ($1, $2, $3, $4, $5)`
	//TODO: Named Exec?
	// entity := types.Entity{
	// 	ID: entityID,
	// 	EntityInfo: types.EntityInfo{
	// 		Address:                 address,
	// 		Email:                   email,
	// 		Name:                    name,
	// 		CensusManagersAddresses: censusManagersAddresses,
	// 	},
	// }
	// insert := `INSERT INTO entities
	// 				(id, address, email, name, census_managers_addresses)
	// 				VALUES (:id, :entityinfo.address, :entityinfo.email, :entityinfo.name, :entityinfo.census_managers_addresses)`
	// _, err := d.db.NamedExec(insert, entity)
	_, err := d.db.Exec(insert, entityID, address, email, name, pq.Array(censusManagersAddresses))
	if err != nil {
		log.Fatal(err)
	}
	entity := types.Entity{}
	selectQuery := "SELECT id, address, email, name, census_managers_addresses FROM entities WHERE id=$1"
	row := d.db.QueryRowx(selectQuery, entityID)
	var addr pq.StringArray
	err = row.Scan(&entity.ID, &entity.Address, &entity.Email, &entity.Name, &addr)
	//TODO: Parse directly struct?
	//err = row.StructScan(&entity1)
	entity.CensusManagersAddresses = addr
	log.Debugf("Addresses: %s", addr)
	js, _ := json.Marshal(entity)
	log.Debugf("%s", string(js))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &entity, nil
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

func (d *Database) CreateMember(entityID string) (*types.Member, error) {
	member := types.Member{}
	js, _ := json.Marshal(member)
	log.Debugf("%s", string(js))
	member.EntityID = "0x12345123451234"
	js, _ = json.Marshal(member)
	log.Debugf("%s", string(js))
	// stmt, err := d.db.Prepare(`INSERT INTO members
	// 				(id, address, email, name, census_managers_addresses)
	// 				VALUES ($1, $2, $3, $4, $5)`)
	return &member, nil
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
