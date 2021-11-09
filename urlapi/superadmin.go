package urlapi

import (
	"fmt"

	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
)

func (u *URLAPI) EnableSuperadminHandlers() error {
	if err := u.api.RegisterMethod(
		"/admin/accounts",
		"POST",
		bearerstdapi.MethodAccessTypeAdmin,
		u.createIntegratorAccountHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/#id",
		"PUT",
		bearerstdapi.MethodAccessTypeAdmin,
		u.updateIntegratorAccountHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/#id/key",
		"PATCH",
		bearerstdapi.MethodAccessTypeAdmin,
		u.resetIntegratorKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/#id",
		"GET",
		bearerstdapi.MethodAccessTypeAdmin,
		u.getIntegratorAccountHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/#id",
		"DELETE",
		bearerstdapi.MethodAccessTypeAdmin,
		u.deleteIntegratorAccountHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/admin/accounts
// createIntegratorAccountHandler creates a new integrator account
func (u *URLAPI) createIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// PUT https://server/v1/admin/accounts/<id>
// updateIntegratorAccountHandler updates an existing integrator account
func (u *URLAPI) updateIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// PATCH https://server/v1/admin/accounts/<id>/key
// resetIntegratorKeyHandler resets an integrator api key
func (u *URLAPI) resetIntegratorKeyHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/admin/accounts/<id>
// getIntegratorAccountHandler fetches an integrator account
func (u *URLAPI) getIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// DELETE https://server/v1/admin/accounts/<id>
// deleteIntegratorAccountHandler deletes an integrator account
func (u *URLAPI) deleteIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}
