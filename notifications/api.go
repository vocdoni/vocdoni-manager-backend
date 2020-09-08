package notifications

import (
	"fmt"

	"firebase.google.com/go/auth"
	"firebase.google.com/go/v4/errorutils"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/metrics"
	"gitlab.com/vocdoni/go-dvote/net"
	"gitlab.com/vocdoni/manager/manager-backend/router"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

// NotificationAPI wraps the push notifications API
type NotificationAPI struct {
	Router *router.Router
	pn     PushNotifier
	ma     *metrics.Agent
}

// NewNotificationAPI creates a new push notifications handler for the Router
func NewNotificationAPI(r *router.Router, pn PushNotifier, ma *metrics.Agent) *NotificationAPI {
	switch pn.(type) {
	case FirebaseAdmin:
		pn.Init()
	}
	return &NotificationAPI{Router: r, pn: pn, ma: ma}
}

// RegisterMethods registers all registry methods behind the given path
func (n *NotificationAPI) RegisterMethods(path string) error {
	var transport net.Transport
	if t, ok := n.Router.Transports["http"]; ok {
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

func (n *NotificationAPI) send(req *router.RouterRequest, resp *types.MetaResponse) {
	if req == nil || req.MessageContext == nil || resp == nil {
		log.Errorf("message context or request is nil, cannot send reply message")
		return
	}
	req.Send(n.Router.BuildReply(req, resp))
}

func (n *NotificationAPI) register(request router.RouterRequest) {
	var response types.MetaResponse
	var createdUser interface{}
	var err error

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
		log.Warnf("invalid public key: %s", request.SignaturePublicKey)
		n.Router.SendError(request, "invalid public key")
		return
	}

	switch n.pn.(type) {
	case FirebaseAdmin:
		var fUser *auth.UserRecord
		// check user
		if u, err := n.pn.GetUser(request.SignaturePublicKey); err != nil {
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
					UID(request.SignaturePublicKey).
					Disabled(false)
				// make firebase request
				createdUser, err = n.pn.CreateUser(params)
				if err != nil {
					log.Warnf("cannot create user: %s", err)
					n.Router.SendError(request, fmt.Sprintf("cannot create user: %s", err))
					return
				}
				fUser = createdUser.(*auth.UserRecord)
				log.Debugf("created new user with uid: %s", fUser.UID)
			}
		} else {
			// found user
			fUser = u.(*auth.UserRecord)
			if fUser.UID != request.SignaturePublicKey {
				log.Warnf("cannot register user, uid and signature mismatch. uid: %s pubkey: %s", fUser.UID, request.SignaturePublicKey)
				n.Router.SendError(request, fmt.Sprintf("cannot register user, uid and signature mismatch. uid: %s pubkey: %s", fUser.UID, request.SignaturePublicKey))
				return
			}
		}
		// generate token
		response.Token, err = n.pn.GenerateToken(fUser.UID)
		if err != nil {
			log.Warnf("cannot generate token: %s", err)
			n.Router.SendError(request, fmt.Sprintf("cannot generate token: %s", err))
			return
		}
		// send successful response
		log.Infof("user: %s generated token is: %s", fUser.UID, response.Token)
		n.send(&request, &response)
		return
	}
	log.Fatal("not supported push notifier")
	n.Router.SendError(request, "internal server error")
}
