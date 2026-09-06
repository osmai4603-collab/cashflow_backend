package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the server, database, and infrastructure runtime configuration.
type Config struct {
	// Server Network & Lifecycle Timeouts
	Port              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	DrainDuration     time.Duration
	ShutdownTimeout   time.Duration
	MaxHeaderBytes    int

	// Storage & Database
	StorageDriver string // "postgres" or "memory"
	DatabaseURL   string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string

	// Application Metadata & ERP Mode
	AppName     string
	AppVersion  string
	CompanyMode string // "single" or "multi"

	// Cache & Redis
	RedisHost string
	RedisPort string

	// Authentication & Security
	JWTSecret string

	// Database Connection Pool
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration
}

// Load reads configuration from environment variables or applies production-safe defaults
// according to the go-server-lifecycle specification.
func Load() *Config {
	port := getEnv("PORT", "8070")

	readTimeoutSec := getEnvInt("READ_TIMEOUT_SECONDS", 5)
	readHeaderTimeoutSec := getEnvInt("READ_HEADER_TIMEOUT_SECONDS", 2)
	writeTimeoutSec := getEnvInt("WRITE_TIMEOUT_SECONDS", 10)
	idleTimeoutSec := getEnvInt("IDLE_TIMEOUT_SECONDS", 120)

	drainSec := getEnvInt("DRAIN_SECONDS", 5)
	shutdownSec := getEnvInt("SHUTDOWN_SECONDS", 10)

	// Storage & PostgreSQL configuration
	storageDriver := getEnv("STORAGE_DRIVER", "postgres")
	dbURL := getEnv("DATABASE_URL", "")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "cashflow")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	// Application & ERP mode
	appName := getEnv("APP_NAME", "odoo_go_backend")
	appVersion := getEnv("APP_VERSION", "0.1.0")
	companyMode := getEnv("COMPANY_MODE", "single")

	// Redis & Cache
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")

	// JWT Authentication
	jwtSecret := getEnv("JWT_SECRET", "odoo-go-insecure-dev-secret-key-change-in-production")

	dbMaxConns := int32(getEnvInt("DB_MAX_CONNS", 25))
	dbMinConns := int32(getEnvInt("DB_MIN_CONNS", 5))
	dbMaxConnLifetimeMin := getEnvInt("DB_MAX_CONN_LIFETIME_MINUTES", 60)
	dbMaxConnIdleTimeMin := getEnvInt("DB_MAX_CONN_IDLE_TIME_MINUTES", 30)

	return &Config{
		Port:              port,
		ReadTimeout:       time.Duration(readTimeoutSec) * time.Second,
		ReadHeaderTimeout: time.Duration(readHeaderTimeoutSec) * time.Second,
		WriteTimeout:      time.Duration(writeTimeoutSec) * time.Second,
		IdleTimeout:       time.Duration(idleTimeoutSec) * time.Second,
		DrainDuration:     time.Duration(drainSec) * time.Second,
		ShutdownTimeout:   time.Duration(shutdownSec) * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB

		StorageDriver:     storageDriver,
		DatabaseURL:       dbURL,
		DBHost:            dbHost,
		DBPort:            dbPort,
		DBUser:            dbUser,
		DBPassword:        dbPassword,
		DBName:            dbName,
		DBSSLMode:         dbSSLMode,

		AppName:           appName,
		AppVersion:        appVersion,
		CompanyMode:       companyMode,
		RedisHost:         redisHost,
		RedisPort:         redisPort,
		JWTSecret:         jwtSecret,

		DBMaxConns:        dbMaxConns,
		DBMinConns:        dbMinConns,
		DBMaxConnLifetime: time.Duration(dbMaxConnLifetimeMin) * time.Minute,
		DBMaxConnIdleTime: time.Duration(dbMaxConnIdleTimeMin) * time.Minute,
	}
}

// RedisAddr returns the formatted Redis host:port address.
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

// DSN returns the PostgreSQL connection string.
// If DATABASE_URL is set, it is returned directly; otherwise, a connection string is formatted from individual parameters.
func (c *Config) DSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
