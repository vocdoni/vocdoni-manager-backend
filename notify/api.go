package notify

import (
	"fmt"

	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/errorutils"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/types"
	"go.vocdoni.io/manager/util"
)

// API wraps the push notifications API
type API struct {
	PushNotifier PushNotifier
}

// NewAPI creates a new push notifications handler for the Router
func NewAPI(pn PushNotifier) *API {
	return &API{PushNotifier: pn}
}

// The register method creates a Firebase token for a user. If the user does not exist
// a new Firebase user is created, else the token is created with the existing user UID
// which is the pubkey extracted from the request signature
func (n *API) Register(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var response types.MetaResponse
	var u User
	var err error
	var signaturePubKey []byte

	if signaturePubKey, err = util.RetrieveSignaturePubKey(ctx); err != nil {
		return err
	}

	// check user
	if u, err = n.PushNotifier.GetUser(string(signaturePubKey)); err != nil {
		// firebase specific
		if n.PushNotifier.Service() == Firebase {
			// if err is user not found continue, else return error
			if errorutils.IsInternal(err) || errorutils.IsUnknown(err) {
				return fmt.Errorf("cannot get user: %s", err)
			}
			if errorutils.IsNotFound(err) {
				// create if user does not exist
				// set info
				params := (&auth.UserToCreate{}).
					UID(string(signaturePubKey)).
					Disabled(false)
				// make firebase request
				u, err = n.PushNotifier.CreateUser(FirebaseUser{UserToCreate: params})
				if err != nil {
					return fmt.Errorf("cannot create user: %s", err)
				}
				log.Debugf("created new user with uid: %s", u.UID())
			}
		} else {
			return fmt.Errorf("unsupported push notification service: %s", err)
		}
	} else {
		// found user
		if u.UID() != string(signaturePubKey) {
			return fmt.Errorf("cannot register user, uid and signature mismatch. uid: %s pubkey: %s", u.UID(), signaturePubKey)
		}
	}
	// generate token
	response.Token, err = n.PushNotifier.GenerateToken(u.UID())
	if err != nil {
		return fmt.Errorf("cannot generate token: %s", err)
	}
	// send successful response
	log.Infof("user: %s generated token is: %s", u.UID(), response.Token)
	return util.SendResponse(response, ctx)
}
