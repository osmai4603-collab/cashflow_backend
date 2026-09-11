package config

import (
	"flag"
	"fmt"
)

// Flags describes the CLI options that can override the configuration, mirroring
// Odoo's command-line layer (option groups HTTP service, database, modules).
type Flags struct {
	// Config overrides the default config file path (see WithFile).
	Config    string
	SaveConfig bool

	// Server
	HTTPInterface string
	HTTPPort      string
	NoHTTP        bool
	ProxyMode     bool

	// Database
	Database    string // -d / --db-name
	DBUser      string
	DBPassword  string
	DBHost      string
	DBPort      string
	DBFilter    string
	ListDatabases bool

	// Modules / runtime (Odoo: --load, -i, -u, --dev, --stop-after-init)
	ServerWideModules string
	InitModules       string
	UpdateModules     string
	DevMode           string
	StopAfterInit     bool

	// Logging / workers / tests
	LogLevel   string
	Workers    int
	TestEnable bool
	TestTags   string
}

// ParseFlags parses the process arguments (excluding the binary name) using the
// standard flag package. It returns an error for unknown flags or missing
// values. When ShowHelp is set the error wraps flag.ErrHelp so callers can print
// usage.
func ParseFlags(args []string) (*Flags, error) {
	var f Flags
	fs := flag.NewFlagSet("cashflow", flag.ContinueOnError)
	fs.StringVar(&f.Config, "config", "", "config file path (overrides $CASHFLOW_RC / config/cashflow.json)")
	fs.BoolVar(&f.SaveConfig, "save", false, "save the effective configuration back to the config file and exit")
	fs.StringVar(&f.HTTPInterface, "http-interface", "", "listen interface (Odoo --http-interface)")
	fs.StringVar(&f.HTTPPort, "http-port", "", "listen port (Odoo --http-port)")
	fs.StringVar(&f.HTTPPort, "p", "", "shorthand for --http-port")
	fs.BoolVar(&f.NoHTTP, "no-http", false, "disable the HTTP server (Odoo --no-http)")
	fs.BoolVar(&f.ProxyMode, "proxy-mode", false, "run behind a reverse proxy (Odoo --proxy-mode)")
	fs.StringVar(&f.Database, "d", "", "target database name (Odoo -d/--db-name)")
	fs.StringVar(&f.DBUser, "db-user", "", "database user")
	fs.StringVar(&f.DBPassword, "db-password", "", "database password")
	fs.StringVar(&f.DBHost, "db-host", "", "database host")
	fs.StringVar(&f.DBPort, "db-port", "", "database port")
	fs.StringVar(&f.DBFilter, "db-filter", "", "database filtering regexp (Odoo --db-filter)")
	fs.BoolVar(&f.ListDatabases, "no-database-list", false, "disable the database listing (Odoo --no-database-list)")
	fs.StringVar(&f.ServerWideModules, "load", "", "comma-separated server-wide modules (Odoo --load)")
	fs.StringVar(&f.InitModules, "i", "", "comma-separated modules to install (Odoo -i/--init)")
	fs.StringVar(&f.InitModules, "init", "", "shorthand for -i")
	fs.StringVar(&f.UpdateModules, "u", "", "comma-separated modules to update (Odoo -u/--update)")
	fs.StringVar(&f.UpdateModules, "update", "", "shorthand for -u")
	fs.StringVar(&f.DevMode, "dev", "", "development mode (relative path or 'all')")
	fs.BoolVar(&f.StopAfterInit, "stop-after-init", false, "stop after initializing modules (Odoo --stop-after-init)")
	fs.StringVar(&f.LogLevel, "log-level", "", "logging level (info, debug, warn, error)")
	fs.IntVar(&f.Workers, "workers", 0, "number of worker goroutines (Odoo --workers; 0 disables the pool)")
	fs.BoolVar(&f.TestEnable, "test-enable", false, "enable the test framework (Odoo --test-enable)")
	fs.StringVar(&f.TestTags, "test-tags", "", "comma-separated test tags (Odoo --test-tags)")
	fs.Usage = func() { fmt.Fprintln(fs.Output(), "usage: cashflow [-config file] [-save] [server|database|module options]") }
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	return &f, nil
}

// Apply mutates the configuration with the parsed CLI layer (highest priority
// after runtime overrides).
func (f *Flags) Apply(cfg *Configuration) {
	if f.HTTPInterface != "" {
		cfg.Server.Interface = f.HTTPInterface
	}
	if f.HTTPPort != "" {
		cfg.Server.Port = f.HTTPPort
	}
	if f.NoHTTP {
		cfg.Server.HTTPEnable = false
	}
	if f.ProxyMode {
		cfg.Server.ProxyMode = true
	}
	if f.Database != "" {
		cfg.Database.Name = f.Database
	}
	if f.DBUser != "" {
		cfg.Database.User = f.DBUser
	}
	if f.DBPassword != "" {
		cfg.Database.Password = f.DBPassword
	}
	if f.DBHost != "" {
		cfg.Database.Host = f.DBHost
	}
	if f.DBPort != "" {
		cfg.Database.Port = f.DBPort
	}
	if f.DBFilter != "" {
		cfg.Database.DBFilter = f.DBFilter
	}
	if f.ListDatabases {
		cfg.Database.ListDB = false
	}
	if f.ServerWideModules != "" {
		cfg.Runtime.ServerWideModules = parseModules(f.ServerWideModules)
	}
	if f.InitModules != "" {
		cfg.Runtime.InitModules = parseModules(f.InitModules)
	}
	if f.UpdateModules != "" {
		cfg.Runtime.UpdateModules = parseModules(f.UpdateModules)
	}
	if f.DevMode != "" {
		cfg.Runtime.DevMode = parseModules(f.DevMode)
	}
	if f.StopAfterInit {
		cfg.Runtime.StopAfterInit = true
	}
	if f.LogLevel != "" {
		cfg.Log.Level = f.LogLevel
	}
	if f.Workers > 0 {
		cfg.Worker.Workers = f.Workers
	}
	if f.TestEnable {
		cfg.Test.Enable = true
	}
	if f.TestTags != "" {
		cfg.Test.Tags = f.TestTags
	}
}

func parseModules(raw string) []string {
	if raw == "" {
		return nil
	}
	return splitModules(raw)
}

func splitModules(raw string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(raw); i++ {
		if i == len(raw) || raw[i] == ',' {
			tok := raw[start:i]
			if tok != "" {
				out = append(out, tok)
			}
			start = i + 1
		}
	}
	return out
}