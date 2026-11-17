package config

import "time"

// Options 是完整的应用配置，从环境变量加载。
type Options struct {
	App
	System
	JWT
	Mysql
	TencentIM
	COS
	SMS
	Redis
	TelegramAlert TelegramAlert
	Turnstile     Turnstile
	WebTurnstile  Turnstile
}

type JWT struct {
	Secret           string
	Expiry           time.Duration
	SuperAdminExpiry time.Duration
	RefreshExpiry    time.Duration
}

type App struct {
	Addr               string
	Port               string
	CORSAllowedOrigins []string
	TrustedProxyCIDRs  []string
	RealIPHeaderOrder  []string
}

var DefaultTrustedProxyCIDRs = []string{
	"127.0.0.1/32",
	"::1/128",
}

var DefaultRealIPHeaderOrder = []string{
	"X-Real-IP",
}

type System struct {
	Logdir   string
	Datadir  string
	LogLevel string
	TimeZone string
}

type Mysql struct {
	Addr     string
	Port     string
	User     string
	Passwd   string
	Database string
	Options  string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

type TencentIM struct {
	SDKAppID                int
	AppKey                  string
	AdminUser               string
	CallbackToken           string
	UserSigExpiry           time.Duration
	UserSigRefreshThreshold time.Duration
}

type COS struct {
	SecretID      string
	SecretKey     string
	Bucket        string
	Region        string
	ChatSecretID  string
	ChatSecretKey string
	ChatBucket    string
	ChatRegion    string
}

type SMS struct {
	Provider        string
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
}

type Redis struct {
	Enabled  bool
	Addr     string
	Password string
	DB       int
}

type TelegramAlert struct {
	Enabled      bool
	BotToken     string
	ChatID       string
	PlatformName string
	DedupeTTL    time.Duration
}

type Turnstile struct {
	Enabled           bool
	SecretKey         string
	VerifyURL         string
	ExpectedHostname  string
	ExpectedHostnames []string
}
