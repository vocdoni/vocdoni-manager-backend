package notify

import (
	"context"
	"errors"
	"fmt"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/vocdoni/go-dvote/chain/ethevents"
	"gitlab.com/vocdoni/go-dvote/log"
	"google.golang.org/api/option"
)

// firebase.go contains all the methods required to interact with the
// FCM Admin SDK wrapped in the FirebaseAdmin struct.
// It also contains the handler functions to deal with ethereum events
// and IPFS files changes.

/* ADMIN */

// FirebaseAdmin wraps the firebase admin SDK App struct and a reference to the IPFS file tracker
type FirebaseAdmin struct {
	*firebase.App
	Client *auth.Client
	Key    string
	IPFS   *IPFSFileTracker
}

// NewFirebaseAdmin returns a pointer to a fresh FirebaseAdmin
func NewFirebaseAdmin(key string, ft *IPFSFileTracker) *FirebaseAdmin {
	return &FirebaseAdmin{Key: key, IPFS: ft}
}

// FirebaseUser wraps a firebase user
type FirebaseUser struct {
	User
	*auth.UserRecord
	*auth.UserToCreate
	*auth.UserToUpdate
	*auth.DeleteUsersResult
}

// UID returns a FirebaseUser UID
func (fu FirebaseUser) UID() string {
	return fu.UserRecord.UID
}

// Service returns an integer representing the Firebase push notifications service
func (fa *FirebaseAdmin) Service() int {
	return Firebase
}

// Init initializes the Firebase Admin instance
func (fa *FirebaseAdmin) Init() (err error) {
	log.Info("initializing Firebase app and client with credentials")
	opt := option.WithCredentialsFile(fa.Key)
	fa.App, err = firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return err
	}
	fa.Client, err = fa.App.Auth(context.Background())
	if err != nil {
		return err
	}
	log.Info("initializing IPFS file tracking service")
	if err := fa.IPFS.Start(); err != nil {
		return err
	}
	// start ipfs handler
	go fa.HandleIPFS()
	log.Info("push notifications ipfs service initialized")
	return nil
}

func (fa *FirebaseAdmin) getMessagingClient() (*messaging.Client, error) {
	return fa.Messaging(context.Background())
}

// subscribe & unsubscribe users

// SubscribeTopic subscribes a list of users to a given topic
func (fa *FirebaseAdmin) SubscribeTopic(tokens []string, topic string) error {
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
func (fa *FirebaseAdmin) UnsubscribeTopic(tokens []string, topic string) error {
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
func (fa *FirebaseAdmin) GetUser(uid string) (User, error) {
	var fu *FirebaseUser
	var err error
	if fu.UserRecord, err = fa.Client.GetUser(context.Background(), uid); err != nil {
		return nil, err
	}
	return fu, nil
}

// GetUserByEmail returns user's data from the user matching the given email
func (fa *FirebaseAdmin) GetUserByEmail(email string) (User, error) {
	var ur *FirebaseUser
	var err error
	if ur.UserRecord, err = fa.Client.GetUserByEmail(context.Background(), email); err != nil {
		return nil, err
	}
	return ur, nil
}

// func (fa *FirebaseAdmin) UserBulk(ids *[]auth.UserIdentifier) (*auth.GetUsersResult, error) {}
// func (fa *FirebaseAdmin) Users() (*auth.UserIterator, error) {}

// CreateUser creates a user with the given user info
func (fa *FirebaseAdmin) CreateUser(userData User) (User, error) {
	var ur *FirebaseUser
	var err error
	if ur.UserRecord, err = fa.Client.CreateUser(context.Background(), userData.(*FirebaseUser).UserToCreate); err != nil {
		return nil, err
	}
	return ur, nil
}

// UpdateUser updates a user given its UID and the info to update
func (fa *FirebaseAdmin) UpdateUser(uid string, userData User) (User, error) {
	var ur *FirebaseUser
	var err error
	if ur.UserRecord, err = fa.Client.UpdateUser(context.Background(), uid, userData.(*FirebaseUser).UserToUpdate); err != nil {
		return nil, err
	}
	return ur, err
}

// DeleteUser deletes a user with the given UID
func (fa *FirebaseAdmin) DeleteUser(uid string) error {
	return fa.Client.DeleteUser(context.Background(), uid)
}

// DeleteUserBulk  deletes a list of users giving its ids
func (fa *FirebaseAdmin) DeleteUserBulk(uids []string) (*auth.DeleteUsersResult, error) {
	return fa.Client.DeleteUsers(context.Background(), uids)
}

// GenerateToken returns a custom Firebase FCM token given an UID
func (fa *FirebaseAdmin) GenerateToken(uid string) (string, error) {
	return fa.Client.CustomToken(context.Background(), uid)
}

// messaging

// Send sends a push notification
func (fa *FirebaseAdmin) Send(pn Notification) error {
	if !fa.Check(pn) {
		return errors.New("invalid push notification")
	}
	fpn := pn.(*FirebasePushNotification)
	switch pn.Platform() {
	case PlatformAndroid:
		if fpn.FCM.Android == nil {
			return errors.New("android config must be set")
		}
	case PlatformIos:
		if fpn.FCM.APNS == nil {
			return errors.New("ios config must be set")
		}
	case PlatformWeb:
		if fpn.FCM.Webpush == nil {
			return errors.New("web config must be set")
		}
	case PlatformAll:
		// continue
	default:
		return errors.New("invalid or unsupported platform")
	}

	return fa.send(fpn)
}

func (fa *FirebaseAdmin) send(pn *FirebasePushNotification) error {
	client, err := fa.getMessagingClient()
	if err != nil {
		return err
	}
	res, err := client.Send(context.Background(), pn.FCM)
	if err != nil {
		return err
	}
	log.Debugf("sent notification: %s", res)
	return nil
}

// Check checks a firebase push notification format
func (fa *FirebaseAdmin) Check(notification Notification) bool {
	if notification.Title() == "" || notification.Body() == "" {
		return false
	}
	return true
}

// handlers

// HandleEthereum handles an Ethereum event
func (fa *FirebaseAdmin) HandleEthereum(event *ethtypes.Log, e *ethevents.EthereumEvents) error {
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

func (fa *FirebaseAdmin) handleEthereumNewProcess(event *ethtypes.Log, e *ethevents.EthereumEvents) (*FirebasePushNotification, error) {
	// get process metadata
	processTx, err := ProcessMeta(&e.ContractABI, event.Data, e.ProcessHandle)
	if err != nil {
		return nil, err
	}
	log.Infof("found new process on Ethereum, pushing notification for PID: %s", processTx.ProcessID)

	// create notification
	// get relevant data
	dataMap := make(map[string]string)
	dataMap["uri"] = fmt.Sprintf("%s/%s/%s", defaultAppRouteNewProcess, processTx.EntityID, processTx.ProcessID)
	dataMap["event"] = defaultEventsName[1]
	dataMap["click_action"] = defaultClickAction
	// add notification fields
	notification := &FirebasePushNotification{}
	notification.FCM.Topic = processTx.EntityID + "_" + defaultLang + defaultVoteTopic
	notification.FCM.Notification.Title = defaultProcessTitle
	notification.FCM.Notification.Body = fmt.Sprintf("Entity %s created a new process", processTx.EntityID)
	dataMap["message"] = notification.FCM.Notification.Body
	notification.FCM.Data = dataMap

	// send notification
	if err := fa.Send(notification); err != nil {
		return nil, err
	}
	return notification, nil
}

// HandleIPFS handles an IPFS file content change
func (fa *FirebaseAdmin) HandleIPFS() {
	for {
		newFeed := <-fa.IPFS.UpdatedFilesQueue
		dataMap := make(map[string]string)
		dataMap["uri"] = fmt.Sprintf("%s/%s", defaultAppRouteNewPost, newFeed.eID)
		dataMap["event"] = defaultEventsName[0]
		dataMap["click_action"] = defaultClickAction
		// add notification fields
		notification := &FirebasePushNotification{}
		notification.FCM.Topic = newFeed.eID + "_" + defaultLang + defaultNewsFeedTopic
		notification.FCM.Notification.Title = defaultNewsFeedTitle
		notification.FCM.Notification.Body = fmt.Sprintf("Entity %s created a new feed", newFeed.eID)
		dataMap["message"] = notification.FCM.Notification.Body
		notification.FCM.Data = dataMap

		// send notification
		if err := fa.Send(notification); err != nil {
			log.Warnf("cannot send notification, error: %s", err)
		}
		log.Infof("notification: %+v sended", notification)
	}
}

/* PUSH NOTIFICATION */

// FirebasePushNotification wraps a FCM notification
type FirebasePushNotification struct {
	Upstream BasePushNotification
	FCM      *messaging.Message
}

// ID returns the base push notification id
func (fpn *FirebasePushNotification) ID() string {
	return fpn.Upstream.ID
}

// Action returns the base push notification action
func (fpn *FirebasePushNotification) Action() string {
	return fpn.Upstream.Action
}

// Body returns the base push notification body
func (fpn *FirebasePushNotification) Body() string {
	return fpn.FCM.Notification.Body
}

// Data returns the base push notification data
func (fpn *FirebasePushNotification) Data() Data {
	return fpn.FCM.Data
}

// Date returns the base push notification date
func (fpn *FirebasePushNotification) Date() time.Time {
	return fpn.Upstream.Date
}

// Image returns the base push notification image
func (fpn *FirebasePushNotification) Image() string {
	return fpn.FCM.Notification.ImageURL
}

// Message returns the base push notification message
func (fpn *FirebasePushNotification) Message() Message {
	return fpn.FCM
}

// Platform returns the base push notification platform
func (fpn *FirebasePushNotification) Platform() int {
	return fpn.Upstream.Platform
}

// Priority returns the base push notification priority
func (fpn *FirebasePushNotification) Priority() string {
	return fpn.Upstream.Priority
}

// Title returns the base push notification title
func (fpn *FirebasePushNotification) Title() string {
	return fpn.FCM.Notification.Title
}

// Token returns the base push notification token
func (fpn *FirebasePushNotification) Token() string {
	return fpn.FCM.Token
}

// Topic returns the base push notification topic
func (fpn *FirebasePushNotification) Topic() string {
	return fpn.FCM.Topic
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
		FCM: &messaging.Message{
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
		FCMOptions: &messaging.WebpushFCMOptions{
			Link: link,
		},
	}
}
