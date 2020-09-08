package notifications

import (
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

const defaultLang = "en"

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
	// user management
	GetUser(uid string) (interface{}, error)
	CreateUser(userData interface{}) (interface{}, error)
	UpdateUser(uid string, userData interface{}) (interface{}, error)
	DeleteUser(uid string) error
	// tokens
	GenerateToken(uid string) (string, error)

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
