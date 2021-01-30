package notify

import (
	"fmt"

	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/errorutils"
	"github.com/vocdoni/multirpc/transports"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/metrics"
	"go.vocdoni.io/manager/router"
	"go.vocdoni.io/manager/types"
)

// API wraps the push notifications API
type API struct {
	Router       *router.Router
	PushNotifier PushNotifier
	MetricsAgent *metrics.Agent
}

// NewAPI creates a new push notifications handler for the Router
func NewAPI(r *router.Router, pn PushNotifier, ma *metrics.Agent) *API {
	return &API{Router: r, PushNotifier: pn, MetricsAgent: ma}
}

// RegisterMethods registers all registry methods behind the given path
func (n *API) RegisterMethods(path string) error {
	var transport transports.Transport
	if t, ok := n.Router.Transports["httpws"]; ok {
		transport = t
	} else if t, ok = n.Router.Transports["http"]; ok {
		transport = t
	} else if t, ok = n.Router.Transports["ws"]; ok {
		transport = t
	} else {
		return fmt.Errorf("no compatible transports found (ws or http)")
	}
	log.Infof("adding namespace notifications %s", path+"/notifications")
	transport.AddNamespace(path + "/notifications")
	if err := n.Router.AddHandler("register", path+"/notifications", n.register, false, false); err != nil {
		return err
	}
	// @jordipainan TODO: n.registerMetrics()
	return nil
}

func (n *API) send(req *router.RouterRequest, resp *types.MetaResponse) {
	if req == nil || req.MessageContext == nil || resp == nil {
		log.Errorf("message context or request is nil, cannot send reply message")
		return
	}
	req.Send(n.Router.BuildReply(req, resp))
}

// The register method creates a Firebase token for a user. If the user does not exist
// a new Firebase user is created, else the token is created with the existing user UID
// which is the pubkey extracted from the request signature
func (n *API) register(request router.RouterRequest) {
	var response types.MetaResponse
	var u User
	var err error

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		n.Router.SendError(request, "invalid public key")
		return
	}

	// check user
	if u, err = n.PushNotifier.GetUser(string(request.SignaturePublicKey)); err != nil {
		// firebase specific
		if n.PushNotifier.Service() == Firebase {
			// if err is user not found continue, else return error
			if errorutils.IsInternal(err) || errorutils.IsUnknown(err) {
				log.Warnf("cannot get user: %s", err)
				n.Router.SendError(request, fmt.Sprintf("cannot get user: %s", err))
				return
			}
			if errorutils.IsNotFound(err) {
				// create if user does not exist
				// set info
				params := (&auth.UserToCreate{}).
					UID(string(request.SignaturePublicKey)).
					Disabled(false)
				// make firebase request
				u, err = n.PushNotifier.CreateUser(FirebaseUser{UserToCreate: params})
				if err != nil {
					log.Warnf("cannot create user: %s", err)
					n.Router.SendError(request, fmt.Sprintf("cannot create user: %s", err))
					return
				}
				log.Debugf("created new user with uid: %s", u.UID())
			}
		} else {
			log.Warnf("unsupported push notification service: %s", err)
			n.Router.SendError(request, fmt.Sprintf("cannot create user: %s", err))
			return
		}
	} else {
		// found user
		if u.UID() != string(request.SignaturePublicKey) {
			log.Warnf("cannot register user, uid and signature mismatch. uid: %s pubkey: %s", u.UID(), request.SignaturePublicKey)
			n.Router.SendError(request, fmt.Sprintf("cannot register user, uid and signature mismatch. uid: %s pubkey: %s", u.UID(), request.SignaturePublicKey))
			return
		}
	}
	// generate token
	response.Token, err = n.PushNotifier.GenerateToken(u.UID())
	if err != nil {
		log.Warnf("cannot generate token: %s", err)
		n.Router.SendError(request, fmt.Sprintf("cannot generate token: %s", err))
		return
	}
	// send successful response
	log.Infof("user: %s generated token is: %s", u.UID(), response.Token)
	n.send(&request, &response)
}
