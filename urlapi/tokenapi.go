package urlapi

import (
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/manager/tokenapi"
)

// RegisterMethods registers all tokenAPI methods behind the given path
func (u *URLAPI) EnableTokenAPIMethods(t *tokenapi.TokenAPI) error {
	if err := u.api.RegisterMethod(
		"/token/revoke",
		"DELETE",
		bearerstdapi.MethodAccessTypePublic,
		t.Revoke,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/token/status",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		t.Status,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/token/generate",
		"PUSH",
		bearerstdapi.MethodAccessTypePublic,
		t.Generate,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/token/importKeysBulk",
		"PATCH",
		bearerstdapi.MethodAccessTypePublic,
		t.ImportKeysBulk,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/token/listKeys",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		t.ListKeys,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/token/deleteKeys",
		"DELETE",
		bearerstdapi.MethodAccessTypePublic,
		t.DeleteKeys,
	); err != nil {
		return err
	}
	return nil
}
