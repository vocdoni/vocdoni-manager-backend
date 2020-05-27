package manager

import (
	"gitlab.com/vocdoni/vocdoni-manager-backend/database"
	"gitlab.com/vocdoni/vocdoni-manager-backend/router"
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
	return nil
}
