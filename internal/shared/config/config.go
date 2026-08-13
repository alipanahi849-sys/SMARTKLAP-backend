package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Environment string    `mapstructure:"environment"`
	Server      Server    `mapstructure:"server"`
	Database    Database  `mapstructure:"database"`
	Redis       Redis     `mapstructure:"redis"`
	JWT         JWT       `mapstructure:"jwt"`
	CORS        CORS      `mapstructure:"cors"`
	RateLimit   RateLimit `mapstructure:"rate_limit"`
	Storage     Storage   `mapstructure:"storage"`
	Realtime    Realtime  `mapstructure:"realtime"`
	SMTP        SMTP      `mapstructure:"smtp"`
	Stripe      Stripe    `mapstructure:"stripe"`
}

// Stripe holds card payment credentials (Stripe).
type Stripe struct {
	SecretKey      string `mapstructure:"secret_key"`
	WebhookSecret  string `mapstructure:"webhook_secret"`
	PublishableKey string `mapstructure:"publishable_key"`
	AppURLScheme   string `mapstructure:"app_url_scheme"`
}

func (s Stripe) Enabled() bool {
	return strings.TrimSpace(s.SecretKey) != ""
}

// SMTP holds outbound mail settings for OTP delivery.
// When Host is empty the app falls back to logging OTPs (dev).
type SMTP struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
	UseTLS   bool   `mapstructure:"use_tls"`
}

func (s SMTP) Enabled() bool {
	return strings.TrimSpace(s.Host) != ""
}

func (s SMTP) Address() string {
	port := strings.TrimSpace(s.Port)
	if port == "" {
		port = "587"
	}
	return fmt.Sprintf("%s:%s", s.Host, port)
}

type Server struct {
	Port           string `mapstructure:"port"`
	Mode           string `mapstructure:"mode"`
	ReadTimeout    int    `mapstructure:"read_timeout"`
	WriteTimeout   int    `mapstructure:"write_timeout"`
	RequestTimeout int    `mapstructure:"request_timeout"`
}

type Database struct {
	Host            string `mapstructure:"host"`
	Port            string `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"db_name"`
	SSLMode         string `mapstructure:"ssl_mode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

type Redis struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWT struct {
	Secret        string `mapstructure:"secret"`
	AccessExpiry  int    `mapstructure:"access_expiry"`
	RefreshExpiry int    `mapstructure:"refresh_expiry"`
	Issuer        string `mapstructure:"issuer"`
	RefreshSecret string `mapstructure:"refresh_secret"`
}

type CORS struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

type RateLimit struct {
	AuthRequests      int `mapstructure:"auth_requests"`
	AuthWindowMinutes int `mapstructure:"auth_window_minutes"`
}

type Storage struct {
	Provider               string `mapstructure:"provider"`
	R2AccountID            string `mapstructure:"r2_account_id"`
	R2AccessKeyID          string `mapstructure:"r2_access_key_id"`
	R2SecretAccessKey      string `mapstructure:"r2_secret_access_key"`
	R2Bucket               string `mapstructure:"r2_bucket"`
	LocalRoot              string `mapstructure:"local_root"`
	LocalPublicURL         string `mapstructure:"local_public_url"`
	MaxAudioFileSizeMB     int    `mapstructure:"max_audio_file_size_mb"`
	MaxVideoFileSizeMB     int    `mapstructure:"max_video_file_size_mb"`
	SignedURLExpirationMin int    `mapstructure:"signed_url_expiration_min"`
	MaxLyricLines          int    `mapstructure:"max_lyric_lines"`
	MaxLyricFileSizeKB     int    `mapstructure:"max_lyric_file_size_kb"`
}

// Realtime holds tunables for the WebSocket delivery layer, the event
// dispatcher, the stale-processing watchdog, and data retention jobs.
//
// All fields have safe production defaults applied by applyRealtimeDefaults so
// that a zero-value Realtime (e.g. in unit tests) still behaves correctly.
type Realtime struct {
	// AllowedOrigins is the WebSocket upgrade Origin allowlist. When empty the
	// layer falls back to CORS.AllowedOrigins. "*" allows any origin.
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	// WSConnectionLimitPerMin limits WebSocket upgrade attempts per IP per minute.
	WSConnectionLimitPerMin int `mapstructure:"ws_connection_limit_per_min"`
	// WSSubscriptionLimitPerMin limits channel subscribe requests per IP per minute.
	WSSubscriptionLimitPerMin int `mapstructure:"ws_subscription_limit_per_min"`
	// StaleProcessingTimeoutMinutes is how long an event may remain in the
	// processing state before the watchdog resets it back to pending.
	StaleProcessingTimeoutMinutes int `mapstructure:"stale_processing_timeout_minutes"`
	// WatchdogIntervalMinutes is how often the stale-processing watchdog runs.
	WatchdogIntervalMinutes int `mapstructure:"watchdog_interval_minutes"`
	// SchedulerEventRetentionDays controls deletion of executed/failed/cancelled
	// scheduler_events rows older than this many days.
	SchedulerEventRetentionDays int `mapstructure:"scheduler_event_retention_days"`
	// RealtimeEventRetentionDays controls deletion of realtime_events rows older
	// than this many days.
	RealtimeEventRetentionDays int `mapstructure:"realtime_event_retention_days"`
	// ShutdownDrainTimeoutSeconds bounds how long graceful shutdown waits for the
	// dispatcher and Hub goroutines to exit before forcing close.
	ShutdownDrainTimeoutSeconds int `mapstructure:"shutdown_drain_timeout_seconds"`
}

var AppConfig *Config

func Load(configPath string) error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("../config")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	AppConfig = &Config{}
	if err := viper.Unmarshal(AppConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set default request timeout if not specified
	if AppConfig.Server.RequestTimeout == 0 {
		AppConfig.Server.RequestTimeout = 30
	}

	applyRealtimeDefaults(AppConfig)

	if err := validateConfig(AppConfig); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

func LoadFromEnv() error {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	AppConfig = &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Server: Server{
			Port:           getEnv("SERVER_PORT", "8080"),
			Mode:           getEnv("SERVER_MODE", "debug"),
			ReadTimeout:    60,
			WriteTimeout:   60,
			RequestTimeout: getEnvAsInt("REQUEST_TIMEOUT_SECONDS", 30),
		},
		Database: Database{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			DBName:          getEnv("DB_NAME", "clap"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: 3600,
		},
		Redis: Redis{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		JWT: JWT{
			Secret:        getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			AccessExpiry:  getEnvAsInt("JWT_ACCESS_EXPIRY", 7200),
			RefreshExpiry: getEnvAsInt("JWT_REFRESH_EXPIRY", 604800),
			Issuer:        getEnv("JWT_ISSUER", "clap"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "your-refresh-secret-key-change-in-production"),
		},
		CORS: CORS{
			AllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ","),
			AllowedMethods: strings.Split(getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"), ","),
			AllowedHeaders: strings.Split(getEnv("CORS_ALLOWED_HEADERS", "Origin,Content-Type,Authorization"), ","),
		},
		RateLimit: RateLimit{
			AuthRequests:      getEnvAsInt("AUTH_RATE_LIMIT_REQUESTS", 5),
			AuthWindowMinutes: getEnvAsInt("AUTH_RATE_LIMIT_WINDOW_MINUTES", 1),
		},
		Storage: Storage{
			Provider:               getEnv("STORAGE_PROVIDER", "local"),
			R2AccountID:            getEnv("R2_ACCOUNT_ID", ""),
			R2AccessKeyID:          getEnv("R2_ACCESS_KEY_ID", ""),
			R2SecretAccessKey:      getEnv("R2_SECRET_ACCESS_KEY", ""),
			R2Bucket:               getEnv("R2_BUCKET", ""),
			LocalRoot:              getEnv("STORAGE_LOCAL_ROOT", "./uploads"),
			LocalPublicURL:         getEnv("STORAGE_LOCAL_PUBLIC_URL", "/uploads"),
			MaxAudioFileSizeMB:     getEnvAsInt("MAX_AUDIO_FILE_SIZE_MB", 20),
			MaxVideoFileSizeMB:     getEnvAsInt("MAX_VIDEO_FILE_SIZE_MB", 50),
			SignedURLExpirationMin: getEnvAsInt("SIGNED_URL_EXPIRATION_MINUTES", 30),
			MaxLyricLines:          getEnvAsInt("MAX_LYRIC_LINES", 5000),
			MaxLyricFileSizeKB:     getEnvAsInt("MAX_LYRIC_FILE_SIZE_KB", 500),
		},
		Realtime: Realtime{
			AllowedOrigins:                splitAndTrim(getEnv("REALTIME_ALLOWED_ORIGINS", "")),
			WSConnectionLimitPerMin:       getEnvAsInt("REALTIME_WS_CONNECTION_LIMIT_PER_MIN", 0),
			WSSubscriptionLimitPerMin:     getEnvAsInt("REALTIME_WS_SUBSCRIPTION_LIMIT_PER_MIN", 0),
			StaleProcessingTimeoutMinutes: getEnvAsInt("REALTIME_STALE_PROCESSING_TIMEOUT_MINUTES", 0),
			WatchdogIntervalMinutes:       getEnvAsInt("REALTIME_WATCHDOG_INTERVAL_MINUTES", 0),
			SchedulerEventRetentionDays:   getEnvAsInt("REALTIME_SCHEDULER_EVENT_RETENTION_DAYS", 0),
			RealtimeEventRetentionDays:    getEnvAsInt("REALTIME_EVENT_RETENTION_DAYS", 0),
			ShutdownDrainTimeoutSeconds:   getEnvAsInt("REALTIME_SHUTDOWN_DRAIN_TIMEOUT_SECONDS", 0),
		},
		SMTP: SMTP{
			Host:     getEnv("SMTP_HOST", "localhost"),
			Port:     getEnv("SMTP_PORT", "1025"),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@clap.local"),
			FromName: getEnv("SMTP_FROM_NAME", "Clap"),
			UseTLS:   getEnvAsBool("SMTP_USE_TLS", false),
		},
		Stripe: Stripe{
			SecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
			PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
			AppURLScheme:   getEnv("STRIPE_APP_URL_SCHEME", "smartklap"),
		},
	}

	applyRealtimeDefaults(AppConfig)

	if err := validateConfig(AppConfig); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

// applyRealtimeDefaults fills in safe production defaults for any unset
// Realtime tunables. It is idempotent and safe to call on a partially
// populated config (e.g. from unit tests).
func applyRealtimeDefaults(cfg *Config) {
	r := &cfg.Realtime
	if r.WSConnectionLimitPerMin <= 0 {
		r.WSConnectionLimitPerMin = 20
	}
	if r.WSSubscriptionLimitPerMin <= 0 {
		r.WSSubscriptionLimitPerMin = 100
	}
	if r.StaleProcessingTimeoutMinutes <= 0 {
		r.StaleProcessingTimeoutMinutes = 5
	}
	if r.WatchdogIntervalMinutes <= 0 {
		r.WatchdogIntervalMinutes = 5
	}
	if r.SchedulerEventRetentionDays <= 0 {
		r.SchedulerEventRetentionDays = 7
	}
	if r.RealtimeEventRetentionDays <= 0 {
		r.RealtimeEventRetentionDays = 7
	}
	if r.ShutdownDrainTimeoutSeconds <= 0 {
		r.ShutdownDrainTimeoutSeconds = 15
	}
}

// splitAndTrim splits a comma-separated string and trims whitespace, dropping
// empty entries. Returns nil for an empty input.
func splitAndTrim(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// IsWSOriginAllowed reports whether the given Origin header value is permitted
// to open a WebSocket connection. An empty origin (non-browser client) is
// always allowed. When no realtime allowlist is configured it falls back to
// the CORS allowlist. "*" matches any origin.
func IsWSOriginAllowed(origin string) bool {
	if AppConfig == nil {
		return true
	}
	if strings.TrimSpace(origin) == "" {
		return true
	}
	allowed := AppConfig.Realtime.AllowedOrigins
	if len(allowed) == 0 {
		allowed = AppConfig.CORS.AllowedOrigins
	}
	for _, a := range allowed {
		if a == "*" || strings.EqualFold(a, origin) {
			return true
		}
	}
	return false
}

func validateConfig(cfg *Config) error {
	if cfg.Database.Password == "" {
		return fmt.Errorf("database password is required")
	}

	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	if len(cfg.JWT.Secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long")
	}

	if cfg.JWT.Secret == "your-secret-key-change-in-production" && cfg.Environment == "production" {
		return fmt.Errorf("JWT secret must be changed from default in production")
	}

	if cfg.JWT.RefreshSecret == "" {
		return fmt.Errorf("JWT refresh secret is required")
	}

	if len(cfg.JWT.RefreshSecret) < 32 {
		return fmt.Errorf("JWT refresh secret must be at least 32 characters long")
	}

	if cfg.JWT.RefreshSecret == "your-refresh-secret-key-change-in-production" && cfg.Environment == "production" {
		return fmt.Errorf("JWT refresh secret must be changed from default in production")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := strings.TrimSpace(strings.ToLower(getEnv(key, "")))
	if valueStr == "" {
		return defaultValue
	}
	switch valueStr {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func GetDSN() string {
	cfg := AppConfig.Database
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)
}

func GetRedisAddr() string {
	cfg := AppConfig.Redis
	return fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
}
