package manager

import (
	"database/sql"

	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/vocdoni-manager-backend/database"
	"gitlab.com/vocdoni/vocdoni-manager-backend/router"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
	"gitlab.com/vocdoni/vocdoni-manager-backend/util"
)

type Manager struct {
	Router *router.Router
	db     database.Database
}

// NewManager creates a new registry handler for the Router
func NewManager(r *router.Router, d database.Database) *Manager {
	return &Manager{Router: r, db: d}
}

// RegisterMethods registers all registry methods behind the given path
func (m *Manager) RegisterMethods(path string) error {
	m.Router.Transport.AddNamespace(path + "/manager")
	if err := m.Router.AddHandler("listMembers", path+"/manager", m.listMembers, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("generateTokens", path+"/manager", m.generateTokens, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("importMembers", path+"/manager", m.importMembers, false); err != nil {
		return err
	}
	return nil
}

func (m *Manager) send(req router.RouterRequest, resp types.ResponseMessage) {
	m.Router.Transport.Send(m.Router.BuildReply(req, resp))
}

func (m *Manager) listMembers(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.ResponseMessage

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.PubKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// check filter
	if err = checkOptions(request.ListOptions); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// Query for members
	if response.Members, err = m.db.ListMembers(entityID, nil, request.ListOptions); err != nil {
		if err == sql.ErrNoRows {
			m.Router.SendError(request, "no members found")
			return
		}
		log.Error(err)
		m.Router.SendError(request, "cannot query for members")
		return
	}
	m.send(request, response)
	log.Info("listMembers")
}

func (m *Manager) generateTokens(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.ResponseMessage

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	if request.Amount < 1 {
		log.Warn("invalid request arguments")
		m.Router.SendError(request, "invalid request arguments")
		return
	}

	response.Tokens = make([]uuid.UUID, request.Amount)
	// TODO: Probably I need to initialize tokens
	err = m.db.CreateMembersWithTokens(entityID, response.Tokens)
	if err != nil {
		log.Error("could not register generated tokens")
		m.Router.SendError(request, "could not register generated tokens")
		return
	}
	log.Infof("generate %d tokens for %s", len(response.Tokens), entityID)
	m.send(request, response)
}

func (m *Manager) importMembers(request router.RouterRequest) {
	log.Info("importMembers")
}

func checkOptions(filter *types.ListOptions) error {
	// Check skip and count
	if filter.Skip < 0 || filter.Count < 0 {
		return fmt.Errorf("invalid filter options")
	}
	// Check sortby
	t := reflect.TypeOf(&types.MemberInfo{})
	if len(filter.SortBy) > 0 {
		_, found := t.FieldByName(strings.Title(filter.SortBy))
		if !found {
			return fmt.Errorf("invalid filter options")
		}
		// Check order
		if len(filter.Order) > 0 && (filter.Order == "asc" || filter.Order == "desc") {
			return fmt.Errorf("invalid filter options")
		}

	}
	// Also check that order does not make sense without sortby
	if len(filter.Order) > 0 {
		return fmt.Errorf("invalid filter options")
	}
	return nil

}
