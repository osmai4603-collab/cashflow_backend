package lifecycle

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

// Config holds all parameters governing the server lifecycle, timeouts,
// listener bindings, health checks, and logging.
type Config struct {
	// Addresses
	PublicHost     string
	PublicPort     string
	ManagementHost string
	ManagementPort string

	// Timeouts
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int

	// Lifecycle Durations
	DrainDuration   time.Duration
	ShutdownTimeout time.Duration

	// Management Security
	ManagementToken string

	// Logging Configuration
	LogColor  bool
	LogToFile bool
	LogDir    string
	LogLevel  string
	AppName   string
}

// DefaultConfig returns production-hardened defaults according to Go standards.
func DefaultConfig() *Config {
	return &Config{
		PublicHost:        "",
		PublicPort:        "8070",
		ManagementHost:    "",
		ManagementPort:    "8066",
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
		DrainDuration:     5 * time.Second,
		ShutdownTimeout:   10 * time.Second,
		ManagementToken:   "",
		LogColor:          false,
		LogToFile:         true,
		LogDir:            "logs",
		LogLevel:          "INFO",
		AppName:           "cashflow",
	}
}

// LoadConfigFromEnv builds Config from environment variables with fallback
// to standard production defaults, supporting CASHFLOW_ prefix and generic names.
func LoadConfigFromEnv() (*Config, error) {
	cfg := DefaultConfig()

	// Public port
	if p := getEnv("CASHFLOW_PORT", "PORT"); p != "" {
		cfg.PublicPort = p
	}
	if h := getEnv("CASHFLOW_HOST", "HOST"); h != "" {
		cfg.PublicHost = h
	}

	// Management port
	if mp := getEnv("CASHFLOW_MGMT_PORT", "MGMT_PORT"); mp != "" {
		cfg.ManagementPort = mp
	}
	if mh := getEnv("CASHFLOW_MGMT_HOST", "MGMT_HOST"); mh != "" {
		cfg.ManagementHost = mh
	}

	// Management Token
	cfg.ManagementToken = getEnv("CASHFLOW_MGMT_TOKEN", "MGMT_TOKEN")

	// Durations
	if d := getEnv("CASHFLOW_DRAIN_DURATION", "DRAIN_DURATION"); d != "" {
		if dur, err := time.ParseDuration(d); err == nil {
			cfg.DrainDuration = dur
		} else if sec, err := strconv.Atoi(d); err == nil {
			cfg.DrainDuration = time.Duration(sec) * time.Second
		}
	}

	if s := getEnv("CASHFLOW_SHUTDOWN_TIMEOUT", "SHUTDOWN_TIMEOUT"); s != "" {
		if dur, err := time.ParseDuration(s); err == nil {
			cfg.ShutdownTimeout = dur
		} else if sec, err := strconv.Atoi(s); err == nil {
			cfg.ShutdownTimeout = time.Duration(sec) * time.Second
		}
	}

	// Logging options
	if c := getEnv("CASHFLOW_LOG_COLOR", "LOG_COLOR"); c == "true" {
		cfg.LogColor = true
	} else if c == "false" {
		cfg.LogColor = false
	} else {
		// Auto-detect TTY
		if stat, err := os.Stderr.Stat(); err == nil {
			cfg.LogColor = (stat.Mode() & os.ModeCharDevice) != 0
		}
	}

	if f := getEnv("CASHFLOW_LOG_TO_FILE", "LOG_TO_FILE"); f == "false" {
		cfg.LogToFile = false
	}

	if dir := getEnv("CASHFLOW_LOG_DIR", "LOG_DIR"); dir != "" {
		cfg.LogDir = dir
	}

	if lvl := getEnv("CASHFLOW_LOG_LEVEL", "LOG_LEVEL"); lvl != "" {
		cfg.LogLevel = lvl
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid lifecycle configuration: %w", err)
	}

	return cfg, nil
}

// PublicAddress formats host and port for public listener.
func (c *Config) PublicAddress() string {
	return net.JoinHostPort(c.PublicHost, c.PublicPort)
}

// ManagementAddress formats host and port for management listener.
func (c *Config) ManagementAddress() string {
	return net.JoinHostPort(c.ManagementHost, c.ManagementPort)
}

// Validate performs fail-fast verification on configuration values.
func (c *Config) Validate() error {
	if c.PublicPort == "" {
		return fmt.Errorf("public port cannot be empty")
	}
	if c.ManagementPort == "" {
		return fmt.Errorf("management port cannot be empty")
	}
	if c.PublicPort == c.ManagementPort && c.PublicHost == c.ManagementHost {
		return fmt.Errorf("public port (%s) and management port (%s) must differ when binding same host",
			c.PublicPort, c.ManagementPort)
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be greater than zero")
	}
	if c.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("read header timeout must be greater than zero")
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be greater than zero")
	}
	if c.IdleTimeout <= 0 {
		return fmt.Errorf("idle timeout must be greater than zero")
	}
	if c.MaxHeaderBytes <= 0 {
		return fmt.Errorf("max header bytes must be greater than zero")
	}
	if c.DrainDuration < 0 {
		return fmt.Errorf("drain duration cannot be negative")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be greater than zero")
	}
	return nil
}

func getEnv(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""
}
