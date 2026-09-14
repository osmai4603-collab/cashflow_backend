package serviceconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// =============================================================================
// Complete 5-Layer Configuration Resolution Pipeline
// =============================================================================

type Configuration struct {
	Server   ServerSettings   `json:"server"`
	Database DatabaseSettings `json:"database"`
	Auth     AuthSettings     `json:"auth"`
}

type ServerSettings struct {
	Port            string   `json:"port" env:"PORT" default:"8080"`
	ReadTimeout     Duration `json:"read_timeout" env:"READ_TIMEOUT" default:"5s"`
	WriteTimeout    Duration `json:"write_timeout" env:"WRITE_TIMEOUT" default:"10s"`
	ShutdownTimeout Duration `json:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT" default:"15s"`
	DrainDuration   Duration `json:"drain_duration" env:"DRAIN_DURATION" default:"5s"`
}

type DatabaseSettings struct {
	Driver   string `json:"driver" env:"DB_DRIVER" default:"postgres"`
	Host     string `json:"host" env:"DB_HOST" default:"localhost"`
	Port     string `json:"port" env:"DB_PORT" default:"5432"`
	Name     string `json:"name" env:"DB_NAME" default:"app_db"`
	User     string `json:"user" env:"DB_USER" default:"app_user"`
	Password string `json:"password" env:"DB_PASSWORD"`
}

type AuthSettings struct {
	JWTSecret string `json:"jwt_secret" env:"JWT_SECRET"`
}

// Defaults returns the hardened fallback configuration.
func Defaults() *Configuration {
	return &Configuration{
		Server: ServerSettings{
			Port:            "8080",
			ReadTimeout:     Duration(5 * time.Second),
			WriteTimeout:    Duration(10 * time.Second),
			ShutdownTimeout: Duration(15 * time.Second),
			DrainDuration:   Duration(5 * time.Second),
		},
		Database: DatabaseSettings{
			Driver: "postgres",
			Host:   "localhost",
			Port:   "5432",
			Name:   "app_db",
			User:   "app_user",
		},
		Auth: AuthSettings{
			JWTSecret: "change-this-default-secret-in-production-now",
		},
	}
}

// Loader executes the layered resolution pipeline.
type Loader struct {
	filePath string
}

func NewLoader(filePath string) *Loader {
	return &Loader{filePath: filePath}
}

// Load compiles the configuration in strict priority order.
func (l *Loader) Load() (*Configuration, error) {
	// Layer 1: Start with hardcoded defaults
	cfg := Defaults()

	// Layer 2: Sparse JSON config file on disk (if present)
	if l.filePath != "" {
		if data, err := os.ReadFile(filepath.Clean(l.filePath)); err == nil && len(data) > 0 {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config file %s: %w", l.filePath, err)
			}
		}
	}

	// Layer 3: Environment Variable overrides
	if val := os.Getenv("PORT"); val != "" {
		cfg.Server.Port = val
	}
	if val := os.Getenv("DB_HOST"); val != "" {
		cfg.Database.Host = val
	}
	if val := os.Getenv("DB_PASSWORD"); val != "" {
		cfg.Database.Password = val
	}
	if val := os.Getenv("JWT_SECRET"); val != "" {
		cfg.Auth.JWTSecret = val
	}

	// Layer 4: Fail-Fast Multi-Error Validation
	validator := NewValidator()
	if err := validator.Validate(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed:\n%w", err)
	}

	return cfg, nil
}
