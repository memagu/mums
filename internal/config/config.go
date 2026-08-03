package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type authConfig struct {
	InviteTokenSize  int
	SessionCleanup   time.Duration
	SessionCookie    string
	SessionTTL       time.Duration
	SessionTokenSize int
}

type dbConfig struct {
	FilePath string
}

type mumsDefaults struct {
	CapacityPerUser         int64
	Currency                string
	DefaultPurchaseQuantity int
	MaxPurchaseQuantity     int
	MinPurchaseQuantity     int
	PriceN0lla              float64
	PricePhadder            float64
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
	Address           string
	CookieSecure      bool
	IdleTimeout       time.Duration
	Origin            string
	ReadHeaderTimeout time.Duration
	SSETimeout        time.Duration
}

type swishConfig struct {
	NumberPattern string
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
			RecentTransactionRuns:   10,
			StepPurchaseQuantity:    1,
		},
		Phaddergrupp: phaddergruppDefaults{
			PrimaryColor:   "#F280A1",
			SecondaryColor: "#9966CC",
		},
	}
	Server serverConfig
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
		InviteTokenSize:  envOrParsed("MUMS_INVITE_TOKEN_SIZE", 32, strconv.Atoi),
		SessionCleanup:   envOrParsed("MUMS_SESSION_CLEANUP", 666*time.Second, time.ParseDuration),
		SessionCookie:    envOr("MUMS_SESSION_COOKIE", "sessionToken"),
		SessionTTL:       envOrParsed("MUMS_SESSION_TTL", 1337*time.Hour, time.ParseDuration),
		SessionTokenSize: envOrParsed("MUMS_SESSION_TOKEN_SIZE", 32, strconv.Atoi),
	}
	DB = dbConfig{
		FilePath: envOr("MUMS_DB_PATH", "mums.sqlite3"),
	}
	Server = serverConfig{
		Address:           envOr("MUMS_ADDRESS", ":11337"),
		CookieSecure:      envOrParsed("MUMS_COOKIE_SECURE", true, strconv.ParseBool),
		IdleTimeout:       envOrParsed("MUMS_IDLE_TIMEOUT", 120*time.Second, time.ParseDuration),
		Origin:            envOr("MUMS_APP_URL", "http://127.0.0.1:11337"),
		ReadHeaderTimeout: envOrParsed("MUMS_READ_HEADER_TIMEOUT", 10*time.Second, time.ParseDuration),
		SSETimeout:        envOrParsed("MUMS_SSE_TIMEOUT", 666*time.Second, time.ParseDuration),
	}
	Swish = swishConfig{
		NumberPattern: envOr("MUMS_SWISH_NUMBER_PATTERN", `^(\+46 ?(\(0\))?|0) ?7[02369]-?\d{3} ?\d{2} ?\d{2}$`),
	}
}
