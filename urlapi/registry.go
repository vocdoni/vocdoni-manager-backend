package urlapi

import (
	"fmt"

	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/manager/registry"
)

func (u *URLAPI) EnableRegistryHandlers(r *registry.Registry) error {
	if r == nil {
		return fmt.Errorf("no registry provided")
	}
	u.registry = r
	if err := u.api.RegisterMethod(
		"/registry/register",
		"POST",
		bearerstdapi.MethodAccessTypePublic,
		u.registry.Register,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/registry/validateToken",
		"POST",
		bearerstdapi.MethodAccessTypePublic,
		u.registry.ValidateToken,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/registry/registrationStatus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.registry.RegistrationStatus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/registry/subscribe",
		"POST",
		bearerstdapi.MethodAccessTypePublic,
		u.registry.Subscribe,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/registry/unsubscribe",
		"PATCH",
		bearerstdapi.MethodAccessTypePublic,
		u.registry.Unsubscribe,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/registry/listSubscriptions",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.registry.ListSubscriptions,
	); err != nil {
		return err
	}
	return nil
}
