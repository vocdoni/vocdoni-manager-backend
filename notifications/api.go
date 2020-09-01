package notifications

import (
	"encoding/hex"
	"fmt"

	"firebase.google.com/go/auth"
	finternal "firebase.google.com/go/v4/internal"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/metrics"
	"gitlab.com/vocdoni/go-dvote/util"
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
	n.Router.Transport.AddNamespace(path + "/notifications")
	if err := n.Router.AddHandler("subscribe", path+"/notifications", n.subscribe, false, false); err != nil {
		return err
	}
	if err := n.Router.AddHandler("unsubscribe", path+"/notifications", n.unsubscribe, false, false); err != nil {
		return err
	}
	// @jordipainan TODO: n.registerMetrics()
	return nil
}

func (n *NotificationAPI) send(req router.RouterRequest, resp types.MetaResponse) {
	n.Router.Transport.Send(n.Router.BuildReply(req, resp))
}

func (n *NotificationAPI) subscribe(request router.RouterRequest) {
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
		log.Warnf("invalid public key: %s", request.SignaturePublicKey)
		n.Router.SendError(request, "invalid public key")
		return
	}

	switch n.pn.(type) {
	case FirebaseAdmin:
		// check user
		if u, err := n.pn.GetUser(request.SignaturePublicKey); err != nil {
			// if err is user not found continue, else return error
			if err.(*finternal.FirebaseError).ErrorCode != finternal.NotFound {
				log.Warnf("cannot get user: %s", err)
				n.Router.SendError(request, fmt.Sprintf("cannot get user: %s", err))
				return
			}
		} else {
			// found user, check pubkey == uid
			if u.(auth.UserRecord).UID == request.SignaturePublicKey {
				log.Warnf("user already exists: %s", err)
				n.Router.SendError(request, "user already exists")
				return
			}
		}
		// create if user does not exist
		// set info
		params := (&auth.UserToCreate{}).
			UID(request.SignaturePublicKey).
			Disabled(false)
		// make firebase request
		createdUser, err := n.pn.CreateUser(params)
		if err != nil {
			log.Warnf("cannot create user: %s", err)
			n.Router.SendError(request, fmt.Sprintf("cannot create user: %s", err))
			return
		}

		// generate token
		response.Token, err = n.pn.GenerateToken(request.SignaturePublicKey)
		if err != nil {
			log.Warnf("cannot generate token: %s", err)
			n.Router.SendError(request, fmt.Sprintf("cannot generate token: %s", err))
			return
		}

		// subscribe to entity
		// check entityID
		entityID, err := hex.DecodeString(util.TrimHex(request.EntityID))
		if err != nil {
			log.Warn(err)
			n.Router.SendError(request, "invalid entityId")
			return
		}
		// @jordipainan TODO: handle language to subscribe by default
		// votes
		if err := n.pn.SubscribeTopic([]string{response.Token}, fmt.Sprintf("/%s/votes", entityID)); err != nil {
			log.Warnf("cannot subscribe to entity: %s topic votes with token: %s. Error: %s", entityID, response.Token, err)
			n.Router.SendError(request, "cannot subscribe to entity")
			return
		}
		// feed
		if err := n.pn.SubscribeTopic([]string{response.Token}, fmt.Sprintf("/%s/feed", entityID)); err != nil {
			log.Warnf("cannot subscribe to entity: %s topic feed with token: %s. Error: %s", entityID, response.Token, err)
			n.Router.SendError(request, "cannot subscribe to entity")
			return
		}

		// send successful response
		log.Infof("user: %s subscribed to entity: %s notifications", createdUser.(auth.UserRecord).UID, request.EntityID)
		n.send(request, response)
	}
}

func (n *NotificationAPI) unsubscribe(req router.RouterRequest) {

}
