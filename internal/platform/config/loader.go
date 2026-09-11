// Package config loads the runtime configuration for the ERP backend.
//
// The loading pipeline mirrors Odoo's layered configuration (defaults < file <
// environment < CLI) with Mattermost-style JSON config files. See
// internal/platform/config for the model shared with api/app layers.
package config

import (
	"fmt"
	"os"
)

// Load returns a fully-populated Configuration built by layering defaults,
// an optional JSON config file, the environment and runtime options. A nil
// filePath or the constant UseDefaultFile selects the default config file.
// When the default file is absent, defaults alone are used.
func Load(opts ...Option) (*Configuration, error) {
	loader := newLoader()
	for _, opt := range opts {
		opt(loader)
	}
	return loader.load()
}

// Option configures the loader pipeline.
type Option func(*loader)

// WithFile sets the explicit config file path; use UseDefaultFile (default) to
// fall back to $CASHFLOW_RC / config/cashflow.json. Use WithNoFile to disable
// file loading entirely.
func WithFile(path string) Option {
	return func(l *loader) { l.filePath = path }
}

// WithNoFile disables config file loading entirely (defaults + env + CLI only).
func WithNoFile() Option {
	return func(l *loader) { l.filePath = NoFile }
}

// WithRuntimeOverrides applies values after env+CLI, matching Odoo's runtime
// layer (config.status/network).
func WithRuntimeOverrides(apply func(*Configuration)) Option {
	return func(l *loader) { l.runtimeOverrides = append(l.runtimeOverrides, apply) }
}

const (
	// UseDefaultFile resolves the config file from CASHFLOW_RC or the default
	// config/cashflow.json path; absence of the file is not an error.
	UseDefaultFile = ""
	// NoFile disables config file loading entirely.
	NoFile = "\x00no-file"
)

// loader executes the layered resolution pipeline.
type loader struct {
	filePath         string
	runtimeOverrides []func(*Configuration)
}

func newLoader() *loader {
	return &loader{filePath: UseDefaultFile}
}

func (l *loader) load() (*Configuration, error) {
	cfg := Defaults()

	// 1st layer: config file (JSON; only present keys override defaults).
	path := l.resolveFilePath()
	if path != "" {
		store, err := NewFileStore(path, false)
		if err != nil {
			return nil, fmt.Errorf("load config file %s: %w", path, err)
		}
		if err := store.LoadInto(cfg); err != nil {
			return nil, fmt.Errorf("read config file %s: %w", path, err)
		}
	}

	// 2nd layer: environment variables (canonical names + PG aliases).
	if err := LoadEnvironment(cfg, os.LookupEnv); err != nil {
		return nil, err
	}

	// 3rd layer: runtime overrides.
	for _, apply := range l.runtimeOverrides {
		apply(cfg)
	}

	// Fail-fast validation before the server boots.
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}

// resolveFilePath implements: explicit flag path > CASHFLOW_RC > default.
func (l *loader) resolveFilePath() string {
	switch {
	case l.filePath == NoFile:
		return ""
	case l.filePath != UseDefaultFile:
		return l.filePath
	}
	if rc, ok := os.LookupEnv("CASHFLOW_RC"); ok && rc != "" {
		return rc
	}
	const defaultPath = "config/cashflow.json"
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}
	return ""
}