package config

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/memagu/mums/pkg/email"
)

type authConfig struct {
	InviteTokenSize        int
	LoginRedirectMaxAge    time.Duration
	PasswordResetTokenSize int
	PasswordResetTTL       time.Duration
	PendingInviteCookie    string
	PendingInviteTTL       time.Duration
	SessionCleanup         time.Duration
	SessionCookie          string
	SessionTTL             time.Duration
	SessionTokenSize       int
	TZCookie               string
}

type dbConfig struct {
	FilePath    string
	BusyTimeout time.Duration
}

type mumsDefaults struct {
	CapacityPerUser         int64
	Currency                string
	DefaultPurchaseQuantity int
	MaxPurchaseQuantity     int
	MinPurchaseQuantity     int
	PriceN0lla              float64
	PricePhadder            float64
	PauseDuration           time.Duration
	RecencyWindowDuration   time.Duration
	RecentTransactionRuns   int
	StepPurchaseQuantity    int
}

type phaddergruppDefaults struct {
	PrimaryColor   string
	SecondaryColor string
}

type defaultsConfig struct {
	Mums         mumsDefaults
	Phaddergrupp phaddergruppDefaults
}

type serverConfig struct {
	Address              string
	CookieSecure         bool
	IdleTimeout          time.Duration
	Origin               string
	ReadHeaderTimeout    time.Duration
	SSEHeartbeatInterval time.Duration
	SSETimeout           time.Duration
}

type swishConfig struct {
	NumberPattern      string
	NumberPatternRegex *regexp.Regexp
}

var (
	Auth     authConfig
	DB       dbConfig
	Defaults = defaultsConfig{
		Mums: mumsDefaults{
			CapacityPerUser:         10,
			Currency:                "SEK",
			DefaultPurchaseQuantity: 5,
			MaxPurchaseQuantity:     10,
			MinPurchaseQuantity:     1,
			PriceN0lla:              10.0,
			PricePhadder:            10.0,
			PauseDuration:           30 * time.Minute,
			RecencyWindowDuration:   time.Hour,
			RecentTransactionRuns:   10,
			StepPurchaseQuantity:    1,
		},
		Phaddergrupp: phaddergruppDefaults{
			PrimaryColor:   "#F280A1",
			SecondaryColor: "#9966CC",
		},
	}
	Server serverConfig
	SMTP   email.SMTPConfig
	Swish  swishConfig
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrParsed[T any](key string, fallback T, parse func(string) (T, error)) T {
	if v := os.Getenv(key); v != "" {
		if parsed, err := parse(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file or error loading:", err)
	}

	Auth = authConfig{
		InviteTokenSize:        envOrParsed("MUMS_INVITE_TOKEN_SIZE", 32, strconv.Atoi),
		LoginRedirectMaxAge:    envOrParsed("MUMS_LOGIN_REDIRECT_MAX_AGE", 6*30*24*time.Hour, time.ParseDuration),
		PasswordResetTokenSize: envOrParsed("MUMS_PASSWORD_RESET_TOKEN_SIZE", 32, strconv.Atoi),
		PasswordResetTTL:       envOrParsed("MUMS_PASSWORD_RESET_TTL", 666*time.Second, time.ParseDuration),
		PendingInviteCookie:    envOr("MUMS_PENDING_INVITE_COOKIE", "pendingInvite"),
		PendingInviteTTL:       envOrParsed("MUMS_PENDING_INVITE_TTL", 69*time.Minute, time.ParseDuration),
		SessionCleanup:         envOrParsed("MUMS_SESSION_CLEANUP", 666*time.Second, time.ParseDuration),
		SessionCookie:          envOr("MUMS_SESSION_COOKIE", "sessionToken"),
		SessionTTL:             envOrParsed("MUMS_SESSION_TTL", 1337*time.Hour, time.ParseDuration),
		SessionTokenSize:       envOrParsed("MUMS_SESSION_TOKEN_SIZE", 32, strconv.Atoi),
		TZCookie:               envOr("MUMS_TZ_COOKIE", "tz"),
	}
	DB = dbConfig{
		FilePath:    envOr("MUMS_DB_PATH", "mums.sqlite3"),
		BusyTimeout: envOrParsed("MUMS_DB_BUSY_TIMEOUT", 5*time.Second, time.ParseDuration),
	}
	Server = serverConfig{
		Address:              envOr("MUMS_ADDRESS", ":11337"),
		CookieSecure:         envOrParsed("MUMS_COOKIE_SECURE", true, strconv.ParseBool),
		IdleTimeout:          envOrParsed("MUMS_IDLE_TIMEOUT", 120*time.Second, time.ParseDuration),
		Origin:               envOr("MUMS_APP_URL", "http://127.0.0.1:11337"),
		ReadHeaderTimeout:    envOrParsed("MUMS_READ_HEADER_TIMEOUT", 10*time.Second, time.ParseDuration),
		SSEHeartbeatInterval: envOrParsed("MUMS_SSE_HEARTBEAT_INTERVAL", 30*time.Second, time.ParseDuration),
		SSETimeout:           envOrParsed("MUMS_SSE_TIMEOUT", 666*time.Second, time.ParseDuration),
	}
	SMTP = email.SMTPConfig{
		From:     envOr("MUMS_SMTP_FROM", ""),
		Host:     envOr("MUMS_SMTP_HOST", ""),
		Password: envOr("MUMS_SMTP_PASSWORD", ""),
		Port:     envOrParsed("MUMS_SMTP_PORT", 587, strconv.Atoi),
		Username: envOr("MUMS_SMTP_USERNAME", ""),
	}
	swishNumberPattern := envOr("MUMS_SWISH_NUMBER_PATTERN", `^(\+46 ?(\(0\))?|0) ?7[02369]-?\d{3} ?\d{2} ?\d{2}$`)
	Swish = swishConfig{
		NumberPattern:      swishNumberPattern,
		NumberPatternRegex: regexp.MustCompile(swishNumberPattern),
	}

	if Server.CookieSecure && strings.HasPrefix(Server.Origin, "http://") {
		log.Println("WARNING: MUMS_COOKIE_SECURE=true with an http:// MUMS_APP_URL — secure cookies will not be sent over plain HTTP")
	}
	if !Server.CookieSecure && strings.HasPrefix(Server.Origin, "https://") {
		log.Println("WARNING: MUMS_COOKIE_SECURE=false with an https:// MUMS_APP_URL — cookies will not be Secure over HTTPS")
	}
}
