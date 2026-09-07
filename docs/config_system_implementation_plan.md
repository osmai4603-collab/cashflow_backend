# نظام التكوين بأسلوب Odoo + Mattermost — الخطة التنفيذية

**المرجع**: [odoo_19_configuration_system_report.md](./odoo_19_configuration_system_report.md) + مراجعة `mattermost/server/config`
**يعتمد على**: البنية التحتية الحالية (`internal/infrastructure/config`)
**التقدير**: ~4-5 أيام عمل
**الأولوية**: 🔴 حرج (أساس كل الأنظمة القادمة)

---

## 1. قرارات التصميم المعتمدة

| البند | القرار |
|-------|--------|
| صيغة الملف | **JSON** مفصول بأقسام (نمط `model.Config` في Mattermost) |
| طبقات الأولوية | `defaults < config file < env vars < CLI flags < runtime` (مطابقة لـ ChainMap في Odoo) |
| تجاوز البيئة | Reflection على الـ struct مع وسوم `env:"DB_HOST"` تحافظ على أسماء البيئة الحالية + aliases مثل `PORT` ومرادفات `PG*` |
| CLI | حزمة `flag` القياسية — `-config`, `-save` + خيارات رئيسية |
| الوصول من api/app | `Configuration` struct في `internal/platform/config` (طبقة مشتركة) + `Loader` في `internal/infrastructure/config` |
| الخصائص غير القابلة للتطبيق | تُعرَّف في الـ struct وتُميَّز بـ `N/A (محجوز)` مع توثيق مفتاح البيئة |

---

## 2. المعمارية والملفات

```
internal/platform/config/          ← Engine + Model (متاح من api و app)
├── config.go                      → Configuration struct (كل الأقسام) + Defaults + Computed methods
├── secret.go                      → PBKDF2-SHA512 (مقابل passlib pbkdf2_sha512)
├── spec.go                        → سجل الخصائص + Validation (مقابل _OdooOption)
├── parse.go                       → تحويلات الأنواع bool/comma/path/duration

internal/infrastructure/config/
├── loader.go                      → Load() — يبني من defaults+file+env+cli+runtime
├── store.go                       → BackingStore: FileStore/MemoryStore (نمط store.go)
├── file.go                        → FileStore JSON + persist 0600 (نمط file.go)
├── environment.go                 → تجاوز البيئة بالـ Reflection + aliases + PG*
├── cli.go                         → أعلام flag + -config + -save
└── validate.go                    → فحص التضارب + aliases + تحذيرات deprecation

config/cashflow.json               ← ملف التكوين الافتراضي
```

## 3. أقسام الـ Configuration ومرجعيتها في التقرير

| القسم | الخصائص المغطاة | المصدر في التقرير |
|-------|----------------|--------------------|
| `Server` | interface, port, http_enable, gevent_port, proxy_mode, x_sendfile, pidfile, data_dir + timeouts | خدمة HTTP + تعدد المعالجات |
| `Database` | host, port, user, password, name, sslmode, app_name, pg_path, replica, maxconns, minconns, template, unaccent, dbfilter, list_db | قاعدة البيانات |
| `App` | name, version, company_mode, default_productivity_apps | Note |
| `Cache` | redis host/port | — |
| `Auth` | jwt_secret | — |
| `Security` | admin_passwd (hash), proxy_access_token, publisher_warranty_url | الأمان |
| `Log` | level, file, syslog, handlers, db, db_level, config | التسجيل |
| `Email` | from, from_filter, smtp_*, ssl cert/key | البريد SMTP |
| `Worker` | workers, max_cron_threads, limit_time_worker_cron, limit_time_* | تعدد المعالجات + Advanced |
| `Limit` | memory soft/hard/(gevent), time_cpu, time_real, time_real_cron, requests | تعدد المعالجات |
| `Runtime` | server_wide_modules, init/update/reinit, with_demo, skip_auto_install, import_partial, stop_after_init, dev_mode, upgrade_path, pre_upgrade_scripts | وحدات النظام |
| `Feature` | import_file_maxbytes/timeout, import_url_regex, csv_internal_sep, reportgz, addons_path | خيارات الملف فقط |
| `Test` | test_enable, test_file, test_tags, screencasts, screenshots | الاختبارات |
| `Transient` | osv_memory_count_limit, transient_age_limit | (N/A محجوز) |
| `GeoIP` | geoip_city_db, geoip_country_db | (محجوز) |
| `I18n` | load_language, i18n_overwrite | الترجمة (N/A) |
| `WebSocket` | keep_alive_timeout, rate_limit_burst/delay | (محجوز) |

## 4. خريطة الخصائص ← أين تُستهلك

| مفتاح Odoo | مفتاح Go | Env | حالة التوظيف |
|---|---|---|---|
| `--http-interface` | `Server.Interface` | `HTTP_INTERFACE` | `server.go` Addr |
| `--http-port` | `Server.Port` | `PORT` | `server.go:47` |
| `--no-http` | `Server.HTTPEnable` | `HTTP_ENABLE` | `server.go` Run |
| `--proxy-mode` | `Server.ProxyMode` | `PROXY_MODE` | `router.go` middleware |
| `--x-sendfile` | `Server.XSendfile` | `X_SENDFILE` | محجوز (attachments) |
| `--gevent-port` | `Server.GeventPort` | `GEVENT_PORT` | N/A |
| `--workers` | `Worker.Workers` | `WORKERS` | `WorkerManager` |
| `--max-cron-threads` | `Worker.MaxCronThreads` | `MAX_CRON_THREADS` | `WorkerManager` |
| `--limit-time-worker-cron` | `Worker.TimeWorkerCron` | `LIMIT_TIME_WORKER_CRON` | عمال cron |
| `--limit-time-real` | `Limit.TimeReal` | `LIMIT_TIME_REAL` | chi Timeout middleware |
| `--limit-time-real-cron` | `Limit.TimeRealCron` | `LIMIT_TIME_REAL_CRON` | عمال |
| `--limit-memory-soft/hard` | `Limit.MemorySoft/Hard` | مطابقة | محجوز |
| `--log-level` | `Log.Level` | `LOG_LEVEL` | `main.go` slog |
| `--logfile` / `--syslog` | `Log.File` / `Log.Syslog` | `LOG_FILE` / `LOG_SYSLOG` | logger |
| `--log-handler` | `Log.Handlers` | `LOG_HANDLER` | logger per-package |
| `db_host` | `Database.Host` | `DB_HOST` / `PGHOST` | DSN/pool |
| `db_port` | `Database.Port` | `DB_PORT` / `PGPORT` | DSN/pool |
| `db_user` | `Database.User` | `DB_USER` / `PGUSER` | DSN/pool |
| `db_password` | `Database.Password` | `DB_PASSWORD` / `PGPASSWORD` | DSN/pool |
| `db_name` | `Database.Name` | `DB_NAME` / `PGDATABASE` | DSN/pool |
| `db_sslmode` | `Database.SSLMode` | `DB_SSLMODE` / `PGSSLMODE` | DSN |
| `db_app_name` | `Database.AppName` | `DB_APP_NAME` | poolCfg |
| `db_maxconn` | `Database.MaxConns` | `DB_MAX_CONNS` | poolCfg |
| `db_replica_*` | `Database.ReplicaHost/Port` | `DB_REPLICA_HOST/PORT` | محجوز |
| `--unaccent` | `Database.Unaccent` | `DB_UNACCENT` | محجوز |
| `admin_passwd` | `Security.AdminHash` | `ADMIN_PASSWD` (hash) | `secret.go` |
| `--dev` | `Runtime.DevMode` | `DEV_MODE` | سجل تفصيلي |
| `--stop-after-init` | `Runtime.StopAfterInit` | `STOP_AFTER_INIT` | `cmd/server` |
| `server_wide_modules` | `Runtime.ServerWideModules` | `SERVER_WIDE_MODULES` | محجوز |
| البقية (import, smtp, i18n, geoip, websocket...) | الأقسام المعنية | وسوم env مقابلة | N/A / محجوز + موثّقة |

## 5. الاختبارات

- `spec_test.go`: الأنواع/الافتراضي/choice/aliases
- `environment_test.go`: reflection + aliases + PG*
- `file_test.go`: persist 0600 + save
- `loader_test.go`: أولوية الطبقات الخمس
- `secret_test.go`: verify/update hash
- `validate_test.go`: تضارب syslog+logfile, dev=all
- إبقاء `cmd server` compile وحقول `cfg.*` الجديدة خضراء