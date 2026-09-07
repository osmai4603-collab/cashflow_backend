// Package config provides the runtime configuration model and engine for the
// ERP backend, inspired by Odoo's layered configuration system
// (odoo.tools.config) and Mattermost's JSON + environment override model.
//
// Priority order (lowest to highest), mirroring Odoo's ChainMap:
//
//	defaults < config file < environment variables < CLI flags < runtime overrides
package config

import (
	"fmt"
	"path/filepath"
	"time"
)

// Configuration is the complete runtime configuration, structured in sections
// following Mattermost's model.Config approach. Every property described in the
// Odoo configuration report is represented here, including those that are not
// yet consumed by the Go backend (marked with an inline "reserved" note).
type Configuration struct {
	Server    ServerSettings    `json:"server,omitempty"`
	Database  DatabaseSettings  `json:"database,omitempty"`
	App       AppSettings       `json:"app,omitempty"`
	Cache     CacheSettings     `json:"cache,omitempty"`
	Auth      AuthSettings      `json:"auth,omitempty"`
	Security  SecuritySettings  `json:"security,omitempty"`
	Log       LogSettings       `json:"log,omitempty"`
	Email     EmailSettings     `json:"email,omitempty"`
	Worker    WorkerSettings    `json:"worker,omitempty"`
	Stock     StockSettings     `json:"stock,omitempty"`
	Limit     LimitSettings     `json:"limit,omitempty"`
	Runtime   RuntimeSettings   `json:"runtime,omitempty"`
	Feature   FeatureSettings   `json:"feature,omitempty"`
	Test      TestSettings      `json:"test,omitempty"`
	Transient TransientSettings `json:"transient,omitempty"`
	GeoIP     GeoIPSettings     `json:"geoip,omitempty"`
	I18n      I18nSettings      `json:"i18n,omitempty"`
	WebSocket WebSocketSettings `json:"websocket,omitempty"`
}

// ServerSettings maps the Odoo HTTP service group plus the go-server-lifecycle
// timeouts.
type ServerSettings struct {
	// Odoo: --http-interface (legacy env HTTP_INTERFACE)
	Interface string `json:"interface" env:"HTTP_INTERFACE"`
	// Odoo: --http-port (legacy env PORT)
	Port string `json:"port" env:"PORT"`
	// Odoo: --no-http
	HTTPEnable bool `json:"http_enable" env:"HTTP_ENABLE"`
	// Odoo: --gevent-port (reserved: no gevent in Go)
	GeventPort int `json:"gevent_port" env:"GEVENT_PORT"`
	// Odoo: --proxy-mode
	ProxyMode bool `json:"proxy_mode" env:"PROXY_MODE"`
	// Odoo: --x-sendfile
	XSendfile bool `json:"x_sendfile" env:"X_SENDFILE"`
	// Odoo: --pidfile
	Pidfile string `json:"pidfile" env:"PIDFILE"`
	// Odoo: data_dir
	DataDir string `json:"data_dir" env:"DATA_DIR"`

	// go-server-lifecycle timeouts (legacy env names kept).
	ReadTimeout       time.Duration `json:"read_timeout" env:"READ_TIMEOUT_SECONDS"`
	ReadHeaderTimeout time.Duration `json:"read_header_timeout" env:"READ_HEADER_TIMEOUT_SECONDS"`
	WriteTimeout      time.Duration `json:"write_timeout" env:"WRITE_TIMEOUT_SECONDS"`
	IdleTimeout       time.Duration `json:"idle_timeout" env:"IDLE_TIMEOUT_SECONDS"`
	DrainDuration     time.Duration `json:"drain_duration" env:"DRAIN_SECONDS"`
	ShutdownTimeout   time.Duration `json:"shutdown_timeout" env:"SHUTDOWN_SECONDS"`
	MaxHeaderBytes    int           `json:"max_header_bytes" env:"MAX_HEADER_BYTES"`
}

// DatabaseSettings maps the Odoo database group (with PG* aliases).
type DatabaseSettings struct {
	// Legacy storage driver selection ("postgres" | "memory").
	StorageDriver string `json:"storage_driver" env:"STORAGE_DRIVER"`
	// Optional full connection string; takes precedence over individual fields in DSN().
	DatabaseURL string `json:"database_url" env:"DATABASE_URL"`

	Host        string `json:"host" env:"DB_HOST"`            // PG alias: PGHOST
	Port        string `json:"port" env:"DB_PORT"`            // PG alias: PGPORT
	User        string `json:"user" env:"DB_USER"`            // PG alias: PGUSER
	Password    string `json:"password" env:"DB_PASSWORD"`    // PG alias: PGPASSWORD
	Name        string `json:"name" env:"DB_NAME"`            // PG alias: PGDATABASE
	SSLMode     string `json:"sslmode" env:"DB_SSLMODE"`      // PG alias: PGSSLMODE
	AppName     string `json:"app_name" env:"DB_APP_NAME"`    // PG alias: PGAPPNAME
	PGPath      string `json:"pg_path" env:"PG_PATH"`         // PG alias: PGPATH
	ReplicaHost string `json:"replica_host" env:"DB_REPLICA_HOST"` // PG alias: PGHOST_REPLICA
	ReplicaPort string `json:"replica_port" env:"DB_REPLICA_PORT"` // PG alias: PGPORT_REPLICA
	Template    string `json:"template" env:"DB_TEMPLATE"`    // PG alias: PGDATABASE_TEMPLATE
	Unaccent    bool   `json:"unaccent" env:"DB_UNACCENT"`

	// Odoo: --db-filter
	DBFilter string `json:"dbfilter" env:"DB_FILTER"`
	// Odoo: --no-database-list inverse (true = list DBs allowed)
	ListDB bool `json:"list_db" env:"LIST_DB"`

	MaxConns        int32         `json:"max_conns" env:"DB_MAX_CONNS"`
	MinConns        int32         `json:"min_conns" env:"DB_MIN_CONNS"`
	MaxConnsGevent  int32         `json:"max_conns_gevent" env:"DB_MAXCONN_GEVENT"` // reserved
	MaxConnLifetime time.Duration `json:"max_conn_lifetime" env:"DB_MAX_CONN_LIFETIME_MINUTES"`
	MaxConnIdleTime time.Duration `json:"max_conn_idle_time" env:"DB_MAX_CONN_IDLE_TIME_MINUTES"`
}

// AppSettings maps application metadata and Odoo's default_productivity_apps.
type AppSettings struct {
	Name                    string `json:"name" env:"APP_NAME"`
	Version                 string `json:"version" env:"APP_VERSION"`
	CompanyMode             string `json:"company_mode" env:"COMPANY_MODE"` // single | multi
	DefaultProductivityApps bool   `json:"default_productivity_apps" env:"DEFAULT_PRODUCTIVITY_APPS"`
}

// CacheSettings maps the Redis cache configuration.
type CacheSettings struct {
	RedisHost string `json:"redis_host" env:"REDIS_HOST"`
	RedisPort string `json:"redis_port" env:"REDIS_PORT"`
}

// AuthSettings maps authentication secrets.
type AuthSettings struct {
	JWTSecret string `json:"jwt_secret" env:"JWT_SECRET"`
}

// SecuritySettings maps the Odoo security group. AdminHash stores a PBKDF2-SHA512
// hash instead of plaintext (mirrors passlib pbkdf2_sha512).
type SecuritySettings struct {
	AdminHash            string `json:"admin_passwd" env:"ADMIN_PASSWD"`
	ProxyAccessToken     string `json:"proxy_access_token" env:"PROXY_ACCESS_TOKEN"`        // reserved
	PublisherWarrantyURL string `json:"publisher_warranty_url" env:"PUBLISHER_WARRANTY_URL"` // reserved
}

// LogSettings maps the Odoo logging group.
type LogSettings struct {
	Level    string   `json:"level" env:"LOG_LEVEL"`   // info | debug | warn | error | ...
	File     string   `json:"file" env:"LOG_FILE"`     // Odoo: --logfile
	Syslog   bool     `json:"syslog" env:"LOG_SYSLOG"` // Odoo: --syslog
	Handlers []string `json:"handlers" env:"LOG_HANDLER"` // Odoo: --log-handler
	DB       string   `json:"db" env:"LOG_DB"`         // reserved
	DBLevel  string   `json:"db_level" env:"LOG_DB_LEVEL"`
	Config   string   `json:"config" env:"LOG_CONFIG"` // reserved
}

// EmailSettings maps the Odoo SMTP group. Fully defined but reserved until a
// mailer subsystem is built.
type EmailSettings struct {
	From                 string `json:"from" env:"EMAIL_FROM"`
	FromFilter           string `json:"from_filter" env:"FROM_FILTER"`
	SMTPHost             string `json:"smtp_server" env:"SMTP_SERVER"`
	SMTPPort             int    `json:"smtp_port" env:"SMTP_PORT"`
	SSL                  bool   `json:"smtp_ssl" env:"SMTP_SSL"`
	User                 string `json:"smtp_user" env:"SMTP_USER"`
	Password             string `json:"smtp_password" env:"SMTP_PASSWORD"`
	CertificateFilename  string `json:"smtp_ssl_certificate_filename" env:"SMTP_SSL_CERTIFICATE_FILENAME"`
	PrivateKeyFilename   string `json:"smtp_ssl_private_key_filename" env:"SMTP_SSL_PRIVATE_KEY_FILENAME"`
}

// WorkerSettings maps Odoo's multiprocessing and cron options.
type WorkerSettings struct {
	Workers        int           `json:"workers" env:"WORKERS"`                 // 0 disables the prefork-like pool
	MaxCronThreads int           `json:"max_cron_threads" env:"MAX_CRON_THREADS"` // Odoo: --max-cron-threads
	TimeWorkerCron time.Duration `json:"time_worker_cron" env:"LIMIT_TIME_WORKER_CRON"` // Odoo: --limit-time-worker-cron
}

// StockSettings maps the reorder and landed-cost runtime options
// (Phase 13 — auto reorder + landed costs).
type StockSettings struct {
	// Enable the background reorder-checker worker.
	ReorderEnabled bool `json:"reorder_enabled" env:"STOCK_REORDER_ENABLED"`
	// How often the reorder-checker evaluates active reorder rules.
	ReorderInterval time.Duration `json:"reorder_interval" env:"STOCK_REORDER_INTERVAL"`
}

// LimitSettings maps Odoo's memory/time/request limits.
type LimitSettings struct {
	MemorySoft       int64         `json:"memory_soft" env:"LIMIT_MEMORY_SOFT"`
	MemoryHard       int64         `json:"memory_hard" env:"LIMIT_MEMORY_HARD"`
	MemorySoftGevent int64         `json:"memory_soft_gevent" env:"LIMIT_MEMORY_SOFT_GEVENT"` // reserved
	MemoryHardGevent int64         `json:"memory_hard_gevent" env:"LIMIT_MEMORY_HARD_GEVENT"` // reserved
	TimeCPU          time.Duration `json:"time_cpu" env:"LIMIT_TIME_CPU"`                     // reserved
	TimeReal         time.Duration `json:"time_real" env:"LIMIT_TIME_REAL"`
	TimeRealCron     time.Duration `json:"time_real_cron" env:"LIMIT_TIME_REAL_CRON"`
	Requests         uint64        `json:"requests" env:"LIMIT_REQUEST"` // reserved
}

// RuntimeSettings maps Odoo module/demo/import runtime options.
type RuntimeSettings struct {
	ServerWideModules []string `json:"server_wide_modules" env:"SERVER_WIDE_MODULES"` // Odoo: --load
	InitModules       []string `json:"init_modules" env:"INIT_MODULES"`               // Odoo: --init
	UpdateModules     []string `json:"update_modules" env:"UPDATE_MODULES"`           // Odoo: --update
	ReinitModules     []string `json:"reinit_modules" env:"REINIT_MODULES"`           // Odoo: --reinit
	WithDemo          bool     `json:"with_demo" env:"WITH_DEMO"`                     // Odoo: --with-demo
	SkipAutoInstall   bool     `json:"skip_auto_install" env:"SKIP_AUTO_INSTALL"`     // Odoo: --skip-auto-install
	ImportPartial     string   `json:"import_partial" env:"IMPORT_PARTIAL"`           // Odoo: -P
	StopAfterInit     bool     `json:"stop_after_init" env:"STOP_AFTER_INIT"`         // Odoo: --stop-after-init
	DevMode           []string `json:"dev_mode" env:"DEV_MODE"`                       // Odoo: --dev
	AddonsPath        []string `json:"addons_path" env:"ADDONS_PATH"`                 // reserved
	UpgradePath       []string `json:"upgrade_path" env:"UPGRADE_PATH"`               // reserved
	PreUpgradeScripts []string `json:"pre_upgrade_scripts" env:"PRE_UPGRADE_SCRIPTS"` // reserved
}

// FeatureSettings maps Odoo file-only import/export options.
type FeatureSettings struct {
	ImportMaxBytes int    `json:"import_file_maxbytes" env:"IMPORT_FILE_MAXBYTES"`
	ImportTimeout  int    `json:"import_file_timeout" env:"IMPORT_FILE_TIMEOUT"`
	ImportURLRegex string `json:"import_url_regex" env:"IMPORT_URL_REGEX"`
	CSVInternalSep string `json:"csv_internal_sep" env:"CSV_INTERNAL_SEP"`
	ReportGZ       bool   `json:"reportgz" env:"REPORTGZ"` // reserved
	BinPath        string `json:"bin_path" env:"BIN_PATH"` // reserved
}

// TestSettings maps the Odoo testing group.
type TestSettings struct {
	Enable      bool   `json:"test_enable" env:"TEST_ENABLE"`
	File        string `json:"test_file" env:"TEST_FILE"`
	Tags        string `json:"test_tags" env:"TEST_TAGS"`
	Screencasts string `json:"screencasts" env:"SCREENCASTS"`
	Screenshots string `json:"screenshots" env:"SCREENSHOTS"`
}

// TransientSettings maps the Odoo transient-model limits (reserved in Go).
type TransientSettings struct {
	MaxCount    int     `json:"osv_memory_count_limit" env:"OSV_MEMORY_COUNT_LIMIT"`
	MaxAgeHours float64 `json:"transient_age_limit" env:"TRANSIENT_AGE_LIMIT"`
}

// GeoIPSettings maps the Odoo GeoIP database paths (reserved).
type GeoIPSettings struct {
	CityDB    string `json:"geoip_city_db" env:"GEOIP_CITY_DB"`
	CountryDB string `json:"geoip_country_db" env:"GEOIP_COUNTRY_DB"`
}

// I18nSettings maps the Odoo internationalisation group (reserved).
type I18nSettings struct {
	LoadLanguage string `json:"load_language" env:"LOAD_LANGUAGE"`
	Overwrite    bool   `json:"i18n_overwrite" env:"I18N_OVERWRITE"`
}

// WebSocketSettings maps the Odoo websocket file-only options (reserved for the
// future bus subsystem).
type WebSocketSettings struct {
	KeepAliveTimeout int     `json:"websocket_keep_alive_timeout" env:"WEBSOCKET_KEEP_ALIVE_TIMEOUT"`
	RateLimitBurst   int     `json:"websocket_rate_limit_burst" env:"WEBSOCKET_RATE_LIMIT_BURST"`
	RateLimitDelay   float64 `json:"websocket_rate_limit_delay" env:"WEBSOCKET_RATE_LIMIT_DELAY"`
}

// Defaults returns a fully populated Configuration with production-safe
// defaults matching the odoo_19_configuration_system_report.
func Defaults() *Configuration {
	return &Configuration{
		Server: ServerSettings{
			Interface:         "0.0.0.0",
			Port:              "8070",
			HTTPEnable:        true,
			GeventPort:        8072,
			Pidfile:           "",
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 2 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       120 * time.Second,
			DrainDuration:     5 * time.Second,
			ShutdownTimeout:   10 * time.Second,
			MaxHeaderBytes:    1 << 20,
		},
		Database: DatabaseSettings{
			StorageDriver:   "postgres",
			Host:            "localhost",
			Port:            "5432",
			User:            "postgres",
			Password:        "postgres",
			Name:            "cashflow",
			SSLMode:         "disable",
			AppName:         "odoo-{pid}",
			Template:        "template0",
			ListDB:          true,
			MaxConns:        25,
			MinConns:        5,
			MaxConnLifetime: 60 * time.Minute,
			MaxConnIdleTime: 30 * time.Minute,
		},
		App: AppSettings{
			Name:        "odoo_go_backend",
			Version:     "0.1.0",
			CompanyMode: "single",
		},
		Cache: CacheSettings{
			RedisHost: "localhost",
			RedisPort: "6379",
		},
		Auth: AuthSettings{
			JWTSecret: "odoo-go-insecure-dev-secret-key-change-in-production",
		},
		Security: SecuritySettings{
			PublisherWarrantyURL: "http://services.odoo.com/publisher-warranty/",
		},
		Log: LogSettings{
			Level:    "info",
			DBLevel:  "warning",
			Handlers: []string{":INFO"},
		},
		Email: EmailSettings{
			SMTPHost: "localhost",
			SMTPPort: 25,
		},
		Worker: WorkerSettings{
			MaxCronThreads: 2,
		},
		Stock: StockSettings{
			ReorderEnabled:  false,
			ReorderInterval: 6 * time.Hour,
		},
		Limit: LimitSettings{
			MemorySoft: 2048 * 1024 * 1024,
			MemoryHard: 2560 * 1024 * 1024,
			TimeReal:   120 * time.Second,
			Requests:   1 << 16,
		},
		Runtime: RuntimeSettings{
			ServerWideModules: []string{"base", "rpc", "web"},
		},
		Feature: FeatureSettings{
			ImportMaxBytes: 10 * 1024 * 1024,
			ImportTimeout:  3,
			ImportURLRegex: `^(?:http|https)://`,
			CSVInternalSep: ",",
		},
		Test: TestSettings{
			Screenshots: "/tmp/odoo_tests",
		},
		Transient: TransientSettings{
			MaxAgeHours: 1.0,
		},
		GeoIP: GeoIPSettings{
			CityDB:    "/usr/share/GeoIP/GeoLite2-City.mmdb",
			CountryDB: "/usr/share/GeoIP/GeoLite2-Country.mmdb",
		},
		WebSocket: WebSocketSettings{
			KeepAliveTimeout: 3600,
			RateLimitBurst:   10,
			RateLimitDelay:   0.2,
		},
	}
}

// DSN returns the PostgreSQL connection string. DATABASE_URL takes precedence.
func (c *Configuration) DSN() string {
	if u := c.Database.DatabaseURL; u != "" {
		return u
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.Database.User, c.Database.Password, c.Database.Host, c.Database.Port, c.Database.Name, c.Database.SSLMode)
}

// RedisAddr returns the formatted Redis host:port address.
func (c *Configuration) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Cache.RedisHost, c.Cache.RedisPort)
}

// HTTPAddr returns the full listen address used by the HTTP server.
func (c *Configuration) HTTPAddr() string {
	return fmt.Sprintf("%s:%s", c.Server.Interface, c.Server.Port)
}

// BaseURL returns the externally advertised base URL.
func (c *Configuration) BaseURL() string {
	if c.Server.Interface == "" || c.Server.Interface == "0.0.0.0" {
		return fmt.Sprintf("http://localhost:%s", c.Server.Port)
	}
	return fmt.Sprintf("http://%s:%s", c.Server.Interface, c.Server.Port)
}

// DataDir returns the data directory, defaulting to ./data when unset.
func (c *Configuration) DataDir() string {
	if c.Server.DataDir != "" {
		return c.Server.DataDir
	}
	return filepath.Join(".", "data")
}

// SessionsDir returns the session directory within the data dir (Odoo session_dir).
func (c *Configuration) SessionsDir() string {
	return filepath.Join(c.DataDir(), "sessions")
}

// Filestore returns the filestore directory for the given database (Odoo filestore).
func (c *Configuration) Filestore(dbname string) string {
	return filepath.Join(c.DataDir(), "filestore", dbname)
}