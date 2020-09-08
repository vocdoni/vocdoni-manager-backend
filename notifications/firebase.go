package notifications

import (
	"context"
	"errors"
	"fmt"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"firebase.google.com/go/messaging"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/vocdoni/go-dvote/chain/ethevents"
	"gitlab.com/vocdoni/go-dvote/log"
	"google.golang.org/api/option"
)

/* ADMIN */

// FirebaseAdmin wraps the firebase admin SDK App struct
type FirebaseAdmin struct {
	*firebase.App
	Client *auth.Client
	Key    interface{}
}

// Init initializes the Firebase Admin instance
func (fa FirebaseAdmin) Init() (err error) {
	v, _ := fa.Key.(string)
	opt := option.WithCredentialsFile(v)
	fa.App, err = firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return err
	}
	fa.Client, err = fa.App.Auth(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func (fa FirebaseAdmin) getMessagingClient() (*messaging.Client, error) {
	return fa.Messaging(context.Background())
}

// subscribe & unsubscribe users

// SubscribeTopic subscribes a list of users to a given topic
func (fa FirebaseAdmin) SubscribeTopic(tokens []string, topic string) error {
	client, err := fa.getMessagingClient()
	if err != nil {
		return err
	}
	if _, err := client.SubscribeToTopic(context.Background(), tokens, topic); err != nil {
		return err
	}
	return nil
}

// UnsubscribeTopic unsubscribes a list of users to a given topic
func (fa FirebaseAdmin) UnsubscribeTopic(tokens []string, topic string) error {
	client, err := fa.getMessagingClient()
	if err != nil {
		return err
	}
	if _, err := client.UnsubscribeFromTopic(context.Background(), tokens, topic); err != nil {
		return err
	}
	return nil
}

// user management

// GetUser retrieves user's data
func (fa FirebaseAdmin) GetUser(uid string) (interface{}, error) {
	return fa.Client.GetUser(context.Background(), uid)
}

// GetUserByEmail returns user's data from the user matching the given email
func (fa FirebaseAdmin) GetUserByEmail(email string) (interface{}, error) {
	return fa.Client.GetUserByEmail(context.Background(), email)
}

// func (fa FirebaseAdmin) UserBulk(ids *[]auth.UserIdentifier) (*auth.GetUsersResult, error) {}
// func (fa FirebaseAdmin) Users() (*auth.UserIterator, error) {}

// CreateUser creates a user with the given user info
func (fa FirebaseAdmin) CreateUser(user interface{}) (interface{}, error) {
	return fa.Client.CreateUser(context.Background(), user.(*auth.UserToCreate))
}

// UpdateUser updates a user given its UID and the info to update
func (fa FirebaseAdmin) UpdateUser(uid string, userFields interface{}) (interface{}, error) {
	return fa.Client.UpdateUser(context.Background(), uid, userFields.(*auth.UserToUpdate))
}

// DeleteUser deletes a user with the given UID
func (fa FirebaseAdmin) DeleteUser(uid string) error {
	return fa.Client.DeleteUser(context.Background(), uid)
}

// DeleteUserBulk  deletes a list of users giving its ids
func (fa FirebaseAdmin) DeleteUserBulk(uids []string) (interface{}, error) {
	return fa.Client.DeleteUsers(context.Background(), uids)
}

// tokens
func (fa FirebaseAdmin) GenerateToken(uid string) (string, error) {
	return fa.Client.CustomToken(context.Background(), uid)
}

// messaging

// Send sends a push notification
func (fa FirebaseAdmin) Send(pn interface{}) error {
	if !fa.Check(pn) {
		return errors.New("invalid push notification")
	}
	fpn := pn.(FirebasePushNotification)
	switch fpn.Common.Platform {
	case PlatformAndroid:
		if fpn.Android == nil {
			return errors.New("android config must be set")
		}
	case PlatformIos:
		if fpn.APNS == nil {
			return errors.New("ios config must be set")
		}
	case PlatformWeb:
		if fpn.Webpush == nil {
			return errors.New("web config must be set")
		}
	case PlatformAll:
		break
	default:
		return errors.New("invalid or unsupported platform")
	}

	return fa.send(&fpn)
}

func (fa FirebaseAdmin) send(pn *FirebasePushNotification) error {
	client, err := fa.getMessagingClient()
	if err != nil {
		return err
	}
	if _, err := client.Send(context.Background(), pn.Message); err != nil {
		return err
	}
	return nil
}

// Check checks a firebase push notification format
func (fa FirebaseAdmin) Check(notification interface{}) bool {
	return true
}

// handlers

// HandleEthereum handles an Ethereum event
func (fa FirebaseAdmin) HandleEthereum(event *ethtypes.Log, e *ethevents.EthereumEvents) error {
	var err error
	var notification *FirebasePushNotification
	switch event.Topics[0].Hex() {
	// new process
	case HashLogProcessCreated.Hex():
		notification, err = fa.handleEthereumNewProcess(event, e)
		if err != nil {
			return err
		}
		log.Infof("notification: %+v sended", notification)
		return nil
	// process results published
	case HashLogResultsPublished.Hex():
		var _ resultsPublished
		// stub
		// return nil
	}
	return nil
}

func (fa FirebaseAdmin) handleEthereumNewProcess(event *ethtypes.Log, e *ethevents.EthereumEvents) (*FirebasePushNotification, error) {
	// get process metadata
	processTx, err := ProcessMeta(&e.ContractABI, event.Data, e.ProcessHandle)
	if err != nil {
		return nil, err
	}
	log.Infof("found new process on Ethereum, pushing notification for PID: %s", processTx.ProcessID)

	// create notification
	// get relevant data
	dataMap := make(map[string]string)
	dataMap["processID"] = processTx.ProcessID
	dataMap["processType"] = processTx.ProcessType
	// add notification fields
	notification := &FirebasePushNotification{}
	notification.Data = dataMap
	notification.Topic = processTx.EntityID + "/" + defaultLang + "/votes"
	notification.Notification.Title = "New process created"
	notification.Notification.Body = fmt.Sprintf("Entity %s created a new process, click me for more details", processTx.EntityID)

	// send notification
	if err := fa.Send(notification); err != nil {
		return nil, err
	}
	return notification, nil
}

//func (fa FirebaseAdmin) handleEthereumResultsPublished(event *ethtypes.Log, e *ethevents.EthereumEvents) (*FirebasePushNotification, error) {
//	return nil, nil
//}

// HandleIPFS handles an IPFS file content change
func (fa FirebaseAdmin) HandleIPFS() error {
	return nil
}

func (fa FirebaseAdmin) Enqueue(notification interface{}) error {
	return nil
}
func (fa FirebaseAdmin) Dequeue(notification interface{}) error {
	return nil
}
func (fa FirebaseAdmin) Queue() interface{} {
	return nil
}

/* PUSH NOTIFICATION */

// FirebasePushNotification wraps a FCM notification
type FirebasePushNotification struct {
	Common BasePushNotification
	*messaging.Message
}

// NewFirebasePushNotification returns an initialized message struct with all its data filled
func NewFirebasePushNotification(
	data map[string]string,
	notification *messaging.Notification,
	androidConfig *messaging.AndroidConfig,
	iosConfig *messaging.APNSConfig,
	webpushConfig *messaging.WebpushConfig,
	FCMOpts *messaging.FCMOptions,
	token, topic, condition string) *FirebasePushNotification {

	return &FirebasePushNotification{
		Message: &messaging.Message{
			Data:         data,
			Notification: notification,
			Android:      androidConfig,
			APNS:         iosConfig,
			Webpush:      webpushConfig,
			FCMOptions:   FCMOpts,
			Token:        token,
			Topic:        topic,
			Condition:    condition,
		},
	}
}

// DefaultFirebasePushNotificationAndroidConfig creates an android notification config with default config opts
func DefaultFirebasePushNotificationAndroidConfig(
	data map[string]string,
	notification *messaging.AndroidNotification) *messaging.AndroidConfig {

	return &messaging.AndroidConfig{
		CollapseKey: "",
		Priority:    "normal",
		// TTL duration by default is 4 weeks (2419200 seconds)
		RestrictedPackageName: "",
		Data:                  data,
		Notification:          notification,
		FCMOptions:            new(messaging.AndroidFCMOptions),
	}
}

// DefaultFirebasePushNotificationIosConfig creates an ios notification config with default config opts
func DefaultFirebasePushNotificationIosConfig(
	headers map[string]string,
	payloadAps *messaging.Aps,
	payloadCustom map[string]interface{}) *messaging.APNSConfig {

	return &messaging.APNSConfig{
		Headers: headers,
		Payload: &messaging.APNSPayload{
			Aps:        payloadAps,
			CustomData: payloadCustom,
		},
		FCMOptions: new(messaging.APNSFCMOptions),
	}
}

// DefaultFirebasePushNotificationWebConfig creates a web notification config with default config opts
func DefaultFirebasePushNotificationWebConfig(
	headers, data map[string]string,
	notification *messaging.WebpushNotification,
	link string) *messaging.WebpushConfig {

	return &messaging.WebpushConfig{
		Headers:      headers,
		Data:         data,
		Notification: notification,
		FcmOptions: &messaging.WebpushFcmOptions{
			Link: link,
		},
	}
}
