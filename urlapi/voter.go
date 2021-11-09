package urlapi

import (
	"fmt"

	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
)

func (u *URLAPI) EnableVoterHandlers() error {
	if err := u.api.RegisterMethod(
		"/pub/censuses/#censusId/token",
		"POST",
		bearerstdapi.MethodAccessTypeAdmin,
		u.registerPublicKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/processes/#processId/auth/#signature",
		"GET",
		bearerstdapi.MethodAccessTypeAdmin,
		u.getProcessInfoHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/pub/censuses/<censusId>/token
// registerPublicKeyHandler registers a voter's public key with a census token
func (u *URLAPI) registerPublicKeyHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/pub/processes/<processId>/auth/<signature>
// getProcessInfoHandler gets process info, including private metadata if confidential,
//  checking the voter's signature for inclusion in the census
func (u *URLAPI) getProcessInfoHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// TODO add listProcessesInfoHandler
