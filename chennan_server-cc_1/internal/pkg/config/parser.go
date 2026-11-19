package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Setup 从 .env 文件（若存在）和系统环境变量加载配置。
// 生产环境直接注入环境变量即可，无需 .env 文件。
func Setup() error {
	// 忽略 .env 不存在的错误（生产环境直接用系统 ENV）
	_ = godotenv.Load(".env")

	opts := Options{
		App: App{
			AdminUserResetPassword: envStr("ADMIN_USER_RESET_PASSWORD", ""),
			Addr:                   envStr("APP_ADDR", "127.0.0.1"),
			Port:                   envStr("APP_PORT", "8080"),
			CORSAllowedOrigins: envStrSlice("APP_CORS_ALLOWED_ORIGINS", []string{
				"https://api.ruizhihe.com",
				"https://api.prfgsu.com",
				"https://api.uktjwg.cn",
				"https://rzh35ht.ruizhihe.com",
				"https://*.ruizhihe.com",
				"http://localhost:5173",
				"http://127.0.0.1:5173",
				"http://localhost:5175",
				"http://127.0.0.1:5175",
			}),
			TrustedProxyCIDRs: envStrSlice("APP_TRUSTED_PROXY_CIDRS", DefaultTrustedProxyCIDRs),
			RealIPHeaderOrder: envStrSlice("APP_REAL_IP_HEADER_ORDER", DefaultRealIPHeaderOrder),
		},
		System: System{
			Logdir:   envStr("LOG_DIR", "./logs"),
			Datadir:  envStr("DATA_DIR", "./data"),
			LogLevel: envStr("LOG_LEVEL", "info"),
			TimeZone: envStr("TIMEZONE", "Asia/Shanghai"),
		},
		JWT: JWT{
			Secret:           envStr("JWT_SECRET", ""),
			Expiry:           envDuration("JWT_EXPIRY", 24*time.Hour),
			SuperAdminExpiry: envDuration("JWT_SUPER_ADMIN_EXPIRY", 4320*time.Hour),
			RefreshExpiry:    envDuration("JWT_REFRESH_EXPIRY", 4320*time.Hour),
		},
		Mysql: Mysql{
			Addr:            envStr("MYSQL_ADDR", "localhost"),
			Port:            envStr("MYSQL_PORT", "3306"),
			User:            envStr("MYSQL_USER", "root"),
			Passwd:          envStr("MYSQL_PASSWD", ""),
			Database:        envStr("MYSQL_DATABASE", ""),
			Options:         envStr("MYSQL_OPTIONS", "charset=utf8mb4&parseTime=true&loc=Local&timeout=2s&readTimeout=15s&writeTimeout=15s"),
			MaxOpenConns:    envInt("MYSQL_MAX_OPEN_CONNS", 50),
			MaxIdleConns:    envInt("MYSQL_MAX_IDLE_CONNS", 20),
			ConnMaxIdleTime: envDuration("MYSQL_CONN_MAX_IDLE_TIME", 3*time.Minute),
			ConnMaxLifetime: envDuration("MYSQL_CONN_MAX_LIFETIME", 6*time.Minute),
		},
		TencentIM: TencentIM{
			SDKAppID:                envInt("TENCENT_IM_SDK_APP_ID", 0),
			AppKey:                  envStr("TENCENT_IM_APP_KEY", ""),
			AdminUser:               envStr("TENCENT_IM_ADMIN_USER", "administrator"),
			CallbackToken:           envStr("TENCENT_IM_CALLBACK_TOKEN", ""),
			UserSigExpiry:           envDuration("TENCENT_IM_USER_SIG_EXPIRY", 168*time.Hour),
			UserSigRefreshThreshold: envDuration("TENCENT_IM_USER_SIG_REFRESH_THRESHOLD", 24*time.Hour),
		},
		COS: COS{
			SecretID:      envStr("COS_SECRET_ID", ""),
			SecretKey:     envStr("COS_SECRET_KEY", ""),
			Bucket:        envStr("COS_BUCKET", ""),
			Region:        envStr("COS_REGION", "ap-guangzhou"),
			ChatSecretID:  envStr("COS_CHAT_SECRET_ID", ""),
			ChatSecretKey: envStr("COS_CHAT_SECRET_KEY", ""),
			ChatBucket:    envStr("COS_CHAT_BUCKET", ""),
			ChatRegion:    envStr("COS_CHAT_REGION", ""),
		},
		SMS: SMS{
			Provider:        envStr("SMS_PROVIDER", "dev"),
			AccessKeyID:     envStr("SMS_ACCESS_KEY_ID", ""),
			AccessKeySecret: envStr("SMS_ACCESS_KEY_SECRET", ""),
			SignName:        envStr("SMS_SIGN_NAME", ""),
			TemplateCode:    envStr("SMS_TEMPLATE_CODE", ""),
		},
		Redis: Redis{
			Enabled:  envBool("REDIS_ENABLED", false),
			Addr:     envStr("REDIS_ADDR", "redis:6379"),
			Password: envStr("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_DB", 0),
		},
		TelegramAlert: TelegramAlert{
			Enabled:      envBool("TELEGRAM_ALERT_ENABLED", false),
			BotToken:     envStr("TELEGRAM_BOT_TOKEN", ""),
			ChatID:       envStr("TELEGRAM_CHAT_ID", ""),
			PlatformName: envStr("TELEGRAM_PLATFORM_NAME", ""),
			DedupeTTL:    envDuration("TELEGRAM_ALERT_DEDUPE_TTL", 5*time.Minute),
		},
		Turnstile: Turnstile{
			Enabled:           envBool("TURNSTILE_ENABLED", false),
			SecretKey:         envStr("TURNSTILE_SECRET_KEY", ""),
			VerifyURL:         envStr("TURNSTILE_VERIFY_URL", "https://challenges.cloudflare.com/turnstile/v0/siteverify"),
			ExpectedHostname:  envStr("TURNSTILE_EXPECTED_HOSTNAME", "ht.wuyoubei.com"),
			ExpectedHostnames: envStrSlice("TURNSTILE_EXPECTED_HOSTNAMES", nil),
		},
		WebTurnstile: Turnstile{
			Enabled:           envBool("WEB_TURNSTILE_ENABLED", false),
			SecretKey:         envStr("WEB_TURNSTILE_SECRET_KEY", ""),
			VerifyURL:         envStr("WEB_TURNSTILE_VERIFY_URL", "https://challenges.cloudflare.com/turnstile/v0/siteverify"),
			ExpectedHostname:  envStr("WEB_TURNSTILE_EXPECTED_HOSTNAME", "web.wuyoubei.com"),
			ExpectedHostnames: envStrSlice("WEB_TURNSTILE_EXPECTED_HOSTNAMES", nil),
		},
	}

	Cfg.set(&opts)
	return nil
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func envStrSlice(key string, def []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return def
	}
	return result
}
