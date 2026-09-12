package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// envAliases maps legacy environment names to their canonical env tag value.
// Mirrors Odoo's config.aliases mechanism.
var envAliases = map[string]string{
	"PGHOST":                "DB_HOST",
	"PGPORT":                "DB_PORT",
	"PGUSER":                "DB_USER",
	"PGPASSWORD":            "DB_PASSWORD",
	"PGDATABASE":            "DB_NAME",
	"PGSSLMODE":             "DB_SSLMODE",
	"PGAPPNAME":             "DB_APP_NAME",
	"PGPATH":                "PG_PATH",
	"PGHOST_REPLICA":        "DB_REPLICA_HOST",
	"PGPORT_REPLICA":        "DB_REPLICA_PORT",
	"PGDATABASE_TEMPLATE":   "DB_TEMPLATE",
	"IMPORT_IMAGE_MAXBYTES": "IMPORT_FILE_MAXBYTES",
	"IMPORT_IMAGE_TIMEOUT":  "IMPORT_FILE_TIMEOUT",
	"IMPORT_IMAGE_REGEX":    "IMPORT_URL_REGEX",
}

// EnvTag returns the canonical environment variable name for a struct field, or
// "" when the field has no env tag. Env tags with value "-" are excluded.
func EnvTag(field reflect.StructField) string {
	tag := field.Tag.Get("env")
	if tag == "" || tag == "-" {
		return ""
	}
	return tag
}

// reverseAliases maps every canonical env name to its legacy aliases
// (e.g. "DB_HOST" -> ["PGHOST"]).
func reverseAliases() map[string][]string {
	reverse := make(map[string][]string, len(envAliases))
	for alias, canonical := range envAliases {
		reverse[canonical] = append(reverse[canonical], alias)
	}
	return reverse
}

// EnvValueFor resolves the value for a canonical env name, checking the
// canonical name first and then any legacy aliases (PGHOST for DB_HOST, ...).
func EnvValueFor(lookup func(string) (string, bool), canonical string) (string, bool) {
	if value, ok := lookup(canonical); ok {
		return value, true
	}
	for _, alias := range reverseAliases()[canonical] {
		if value, ok := lookup(alias); ok {
			return value, true
		}
	}
	return "", false
}

// IterateFields walks all tagged fields of Configuration and yields each field
// (direct children of sections, since sections contain the env tags).
func IterateFields(cfg *Configuration, fn func(section string, field reflect.StructField, value reflect.Value)) {
	cfgVal := reflect.ValueOf(cfg).Elem()
	cfgType := cfgVal.Type()
	for i := 0; i < cfgType.NumField(); i++ {
		sectionName := cfgType.Field(i).Name
		section := cfgVal.Field(i)
		if section.Kind() != reflect.Struct {
			continue
		}
		sectionType := section.Type()
		for j := 0; j < sectionType.NumField(); j++ {
			field := sectionType.Field(j)
			if EnvTag(field) == "" {
				continue
			}
			fn(sectionName, field, section.Field(j))
		}
	}
}

// Options collects the tagged fields as a map keyed by env name (spec register).
func Options(cfg *Configuration) map[string]EnvOption {
	options := make(map[string]EnvOption)
	IterateFields(cfg, func(_ string, field reflect.StructField, _ reflect.Value) {
		name := EnvTag(field)
		if name == "" {
			return
		}
		if _, exists := options[name]; exists {
			panic(fmt.Sprintf("duplicate env option %q", name))
		}
		options[name] = EnvOption{Name: name, Type: field.Type}
	})
	return options
}

// EnvOption describes a single configurable option.
type EnvOption struct {
	Name string
	Type reflect.Type
}

// Validate inspects the configuration for cross-field conflicts and invalid
// values. It performs the role of Odoo's _postprocess_options. The returned
// error (if any) should stop server startup.
func (c *Configuration) Validate() error {
	if c.Server.Port != "" {
		if port, err := strconv.Atoi(c.Server.Port); err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid server.port %q: must be a TCP port 1-65535", c.Server.Port)
		}
	}
	switch strings.ToLower(c.App.CompanyMode) {
	case "", "single", "multi":
	default:
		return fmt.Errorf("invalid app.company_mode %q: must be single or multi", c.App.CompanyMode)
	}
	switch strings.ToLower(c.Database.StorageDriver) {
	case "postgres", "memory":
	default:
		return fmt.Errorf("invalid database.storage_driver %q: must be postgres or memory", c.Database.StorageDriver)
	}
	if c.Log.Syslog && c.Log.File != "" {
		return fmt.Errorf("log.syslog and log.file are mutually exclusive")
	}
	if c.Worker.Workers > 0 && c.Worker.MaxCronThreads > c.Worker.Workers {
		return fmt.Errorf("worker.max_cron_threads must not exceed worker.workers")
	}
	if c.Limit.MemoryHard > 0 && c.Limit.MemorySoft > c.Limit.MemoryHard {
		return fmt.Errorf("limit.memory_soft must not exceed limit.memory_hard")
	}
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret must not be empty")
	}
	if err := c.validateManagement(); err != nil {
		return err
	}
	return nil
}

// validateManagement enforces the security posture of the isolated management
// listener:
//   - the port must be a valid TCP port and distinct from the public server
//     port so metrics/pprof are never reachable on the business endpoint;
//   - require_auth without a token is rejected;
//   - binding to 0.0.0.0 / :: is only allowed with auth (never the default).
func (c *Configuration) validateManagement() error {
	if !c.Management.Enabled {
		return nil
	}
	mport, err := strconv.Atoi(c.Management.Port)
	if err != nil || mport < 1 || mport > 65535 {
		return fmt.Errorf("invalid management.port %q: must be a TCP port 1-65535", c.Management.Port)
	}
	if c.Server.Port != "" && c.Management.Port == c.Server.Port {
		return fmt.Errorf("management.port must not equal server.port (%s): metrics/pprof would be exposed on the public endpoint", c.Management.Port)
	}
	switch strings.ToLower(c.Management.Interface) {
	case "0.0.0.0", "::", "":
		if !c.Management.RequireAuth {
			return fmt.Errorf("management.interface %q requires management.require_auth=true (never expose admin/metrics without network isolation)", c.Management.Interface)
		}
	}
	if c.Management.RequireAuth && c.Management.AuthToken == "" {
		return fmt.Errorf("management.require_auth=true requires a non-empty management.auth_token")
	}
	return nil
}
