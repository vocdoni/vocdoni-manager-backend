package notify

import (
	"context"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/vocdoni/go-dvote/chain/ethevents"
)

// lang represents the supported langs by https://www.loc.gov/standards/iso639-2/php/code_list.php
var langs = [...]string{"aa", "ab", "af", "ak", "sq", "am", "ar", "an", "hy", "as", "av", "ae",
	"ay", "az", "ba", "bm", "eu", "be", "bn", "bh", "bi", "bo", "bs", "br", "bg", "my", "ca",
	"cs", "ch", "ce", "zh", "cu", "cv", "kw", "co", "cr", "cy", "cs", "da", "de", "dv", "nl",
	"dz", "el", "en", "eo", "et", "eu", "ee", "fo", "fa", "fj", "fi", "fr", "fr", "fy", "ff",
	"ka", "de", "gd", "ga", "gl", "gv", "el", "gn", "gu", "ht", "ha", "he", "hz", "hi", "ho",
	"hr", "hu", "hy", "ig", "is", "io", "ii", "iu", "ie", "ia", "id", "ik", "is", "it", "jv",
	"ja", "kl", "kn", "ks", "ka", "kr", "kk", "km", "ki", "rw", "ky", "kv", "kg", "ko", "kj",
	"ku", "lo", "la", "lv", "li", "ln", "lt", "lb", "lu", "lg", "mk", "mh", "ml", "mi", "mr",
	"ms", "mk", "mg", "mt", "mn", "mi", "ms", "my", "na", "nv", "nr", "nd", "ng", "ne", "nl",
	"nn", "nb", "no", "ny", "oc", "oj", "or", "om", "os", "pa", "fa", "pi", "pl", "pt", "ps",
	"qu", "rm", "ro", "ro", "rn", "ru", "sg", "sa", "si", "sk", "sk", "sl", "se", "sm", "sn",
	"sd", "so", "st", "es", "sq", "sc", "sr", "ss", "su", "sw", "sv", "ty", "ta", "tt", "te",
	"tg", "tl", "th", "bo", "ti", "to", "tn", "ts", "tk", "tr", "tw", "ug", "uk", "ur", "uz",
	"ve", "vi", "vo", "cy", "wa", "wo", "xh", "yi", "yo", "za", "zh", "zu"}

const (
	defaultLangTag            = "default"
	defaultClickAction        = "FLUTTER_NOTIFICATION_CLICK"
	defaultTopicProcessNew    = "_process-new"
	defaultTopicPostNew       = "_post-new"
	defaultProcessTitle       = "New process created"
	defaultNewsFeedTitle      = "New feed created"
	defaultAppRouteNewProcess = "https://vocdoni.link/processes"
	defaultAppRouteNewPost    = "https://vocdoni.link/posts/view"
)

// supported Push notifications service
const (
	Firebase = iota + 1
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

// User wraps any user type
type User interface {
	UID() string
}

// PushNotifier contains the methos that all push notification services should implement
type PushNotifier interface {
	// Service returns the push notifications selected service
	Service() int
	// topic subscription
	SubscribeTopic(tokens []string, topic string) error
	UnsubscribeTopic(tokens []string, topic string) error
	// messaging
	Check(notification Notification) bool
	Send(notification Notification) error
	// ethereum
	HandleEthereum(ctx context.Context, event *ethtypes.Log, e *ethevents.EthereumEvents) error
	// ipfs
	HandleIPFS()
	// user management
	GetUser(uid string) (User, error)
	CreateUser(userData User) (User, error)
	UpdateUser(uid string, userData User) (User, error)
	DeleteUser(uid string) error
	// tokens
	GenerateToken(uid string) (string, error)

	Init() error
}

// Notification is the interface wrapping the methods that any Notification must implement
type Notification interface {
	ID() string
	Action() string
	Body() string
	Data() Data
	Date() time.Time
	Image() string
	Message() Message
	Platform() int
	Priority() string
	Title() string
	Token() string
	Topic() string
}

// Data represents the notification data
type Data interface{}

// Message represents a notification message
type Message interface{}

// BasePushNotification is a base notification wrapper
type BasePushNotification struct {
	// this are the most common fields among many push notifications services
	ID       string
	Action   string
	Body     string
	Data     Data
	Date     time.Time
	Image    string
	Message  Message
	Platform int
	Priority string
	Title    string
	Token    string
	Topic    string
}
