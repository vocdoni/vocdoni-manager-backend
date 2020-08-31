package notifications

import (
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/vocdoni/go-dvote/chain/ethevents"
)

// message types by receiver
const (
	TopicMessage = iota + 1
	ConditionMessage
	TokenMessage
	GroupMessage
)

// supported platforms
const (
	PlatformAndroid = iota + 1
	PlatformIos
	PlatformWeb
	PlatformAll
)

// PushNotifier contains the methos that all push notification services should implement
type PushNotifier interface {
	// topic subscription
	SubscribeTopic(tokens []string, topic string) error
	UnsubscribeTopic(tokens []string, topic string) error
	// messaging
	Check(notification interface{}) bool
	Send(notification interface{}) error
	// ethereum
	HandleEthereum(event *ethtypes.Log, e *ethevents.EthereumEvents) error
	// ipfs
	HandleIPFS() error
	// notification queue
	Enqueue(notification interface{}) error
	Dequeue(notification interface{}) error
	Queue() interface{}
}

// PushNotifierAdmin contains the methods that all notification push admin services must implement
type PushNotifierAdmin interface {
	// user management
	GetUser(uid string) (interface{}, error)
	CreateUser(uid string, userData interface{}) (interface{}, error)
	UpdateUser(uid string, userData interface{}) (interface{}, error)
	DeleteUser(uid string) error

	Init() error
}

// BasePushNotification is a base notification wrapper
type BasePushNotification struct {
	// this are the most common fields among many push notifications services
	ID       string
	Action   string
	Body     string
	Data     map[string]interface{}
	Date     time.Time
	Image    string
	Message  string
	Platform int
	Priority string
	Title    string
	Token    string
	Topic    string
}
