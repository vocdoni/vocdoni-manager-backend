package urlapi

import (
	"fmt"

	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
)

func (u *URLAPI) EnableEntityHandlers() error {
	if err := u.api.RegisterMethod(
		"/priv/entities/#entityId/metadata",
		"PUT",
		bearerstdapi.MethodAccessTypeAdmin,
		u.setEntityMetadataHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/entities/#entityId/processes/*",
		"POST",
		bearerstdapi.MethodAccessTypeAdmin,
		u.createProcessHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/entities/#entityId/processes/*",
		"GET",
		bearerstdapi.MethodAccessTypeAdmin,
		u.listProcessesHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/processes/#processId",
		"GET",
		bearerstdapi.MethodAccessTypeAdmin,
		u.getProcessHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses",
		"POST",
		bearerstdapi.MethodAccessTypeAdmin,
		u.createCensusHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/#censusId/tokens/*",
		"POST",
		bearerstdapi.MethodAccessTypeAdmin,
		u.addCensusTokensHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/#censusId/tokens/#tokenId",
		"GET",
		bearerstdapi.MethodAccessTypeAdmin,
		u.getCensusTokenHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/#censusId/tokens/#tokenId",
		"DELETE",
		bearerstdapi.MethodAccessTypeAdmin,
		u.deleteCensusTokenHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/#censusId/tokens/#tokenId",
		"DELETE",
		bearerstdapi.MethodAccessTypeAdmin,
		u.deleteCensusTokenHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/#censusId/import/*",
		"POST",
		bearerstdapi.MethodAccessTypeAdmin,
		u.importPublicKeysHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/processes/#processId/status",
		"PUT",
		bearerstdapi.MethodAccessTypeAdmin,
		u.setProcessStatusHandler,
	); err != nil {
		return err
	}
	// HERE STILL REGISTERING ENTITY METHODS< THEN CHECK TO MAKE SURE ALL METHODS ARE REGISTERED CORRECTLY
	return nil
}

// PUT https://server/v1/priv/entities/<entityId>/metadata
// setEntityMetadataHandler sets an entity's metadata
func (u *URLAPI) setEntityMetadataHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// POST https://server/v1/priv/entities/<entityId>/processes/signed
// POST https://server/v1/priv/entities/<entityId>/processes/blind
// createProcessHandler creates a process with the given metadata, either with signed or blind signature voting
func (u *URLAPI) createProcessHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/priv/entities/<entityId>/processes/signed
// GET https://server/v1/priv/entities/<entityId>/processes/blind
// GET https://server/v1/priv/entities/<entityId>/processes/active
// GET https://server/v1/priv/entities/<entityId>/processes/ended
// GET https://server/v1/priv/entities/<entityId>/processes/upcoming
// listProcessesHandler lists signed, blind, active, ended, or upcoming processes
func (u *URLAPI) listProcessesHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/priv/processes/<processId>
// getProcessHandler gets the entirety of a process, including metadata
// confidential processes need no extra step, only the api key
func (u *URLAPI) getProcessHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// POST https://server/v1/priv/censuses
// createCensusHandler creates a census where public keys or token slots (that will eventually contain a public key) are stored.
// A census can start with 0 items, and public keys can be imported later on.
// If census tokens are allocated, users will need to generate a wallet on the frontend and register the public key by themselves.
// This prevents both the API and the integrator from gaining access to the private key.
func (u *URLAPI) createCensusHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// POST https://server/v1/priv/censuses/<censusId>/tokens/flat
// POST https://server/v1/priv/censuses/<censusId>/tokens/weighted
// addCensusTokensHandler adds N (weight 1 or weighted) census tokens for voters to register their public keys
func (u *URLAPI) addCensusTokensHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/priv/censuses/<censusId>/tokens/<tokenId>
// getCensusTokenHandler gets the given census token with weight and assigned public key, if applicable
func (u *URLAPI) getCensusTokenHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// DELETE https://server/v1/priv/censuses/<censusId>/tokens/<tokenId>
// deleteCensusTokenHandler deletes the given token(s) from the given census
func (u *URLAPI) deleteCensusTokenHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// DELETE https://server/v1/priv/censuses/<censusId>/keys/<publicKey>
// deletePublicKeyHandler deletes the given public key(s) from the given census
func (u *URLAPI) deletePublicKeyHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// POST https://server/v1/priv/censuses/<censusId>/import/flat
// POST https://server/v1/priv/censuses/<censusId>/import/weighted
// importPublicKeysHandler imports a group of public keys into the existing census, weighted or weight 1
func (u *URLAPI) importPublicKeysHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// PUT https://server/v1/priv/processes/<processId>/status
// setProcessStatusHandler sets the process status (READY, PAUSED, ENDED, CANCELED)
func (u *URLAPI) setProcessStatusHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}
