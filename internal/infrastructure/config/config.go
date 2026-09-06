package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the server and infrastructure runtime configuration.
type Config struct {
	Port              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	DrainDuration     time.Duration
	ShutdownTimeout   time.Duration
	MaxHeaderBytes    int
}

// Load reads configuration from environment variables or applies production-safe defaults
// according to the go-server-lifecycle specification.
func Load() *Config {
	port := getEnv("PORT", "8080")

	readTimeoutSec := getEnvInt("READ_TIMEOUT_SECONDS", 5)
	readHeaderTimeoutSec := getEnvInt("READ_HEADER_TIMEOUT_SECONDS", 2)
	writeTimeoutSec := getEnvInt("WRITE_TIMEOUT_SECONDS", 10)
	idleTimeoutSec := getEnvInt("IDLE_TIMEOUT_SECONDS", 120)

	drainSec := getEnvInt("DRAIN_SECONDS", 5)
	shutdownSec := getEnvInt("SHUTDOWN_SECONDS", 10)

	return &Config{
		Port:              port,
		ReadTimeout:       time.Duration(readTimeoutSec) * time.Second,
		ReadHeaderTimeout: time.Duration(readHeaderTimeoutSec) * time.Second,
		WriteTimeout:      time.Duration(writeTimeoutSec) * time.Second,
		IdleTimeout:       time.Duration(idleTimeoutSec) * time.Second,
		DrainDuration:     time.Duration(drainSec) * time.Second,
		ShutdownTimeout:   time.Duration(shutdownSec) * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}
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
