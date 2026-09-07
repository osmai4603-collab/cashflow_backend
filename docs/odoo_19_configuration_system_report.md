# نظام ملفات التكوين في Odoo 19.0

> التقرير يوضّح نظام ملفات الـ Configuration في Odoo 19.0، وهو النظام الذي يُفعّل من خلاله ميزات وخصائص النظام، ويُحمَّل أثناء إقلاع السيرفر. مصدر المعلومات: `odoo/tools/config.py`.

## 1. الملف الرئيسي

**`odoo/tools/config.py`** — هذا هو نظام التكوين الأساسي (1079 سطراً). يحتوي على فئة `configmanager` التي تُعدّ كل الخصائص وتُحمّلها أثناء الإقلاع. في نهاية الملف يتم إنشاء الكائن العام:

```python
config = configmanager()   # config.py:1079
```

## 2. ملفات التكوين النموذجية (الافتراضية)

| الملف | الغرض |
| ------- | ------- |
| `debian/odoo.conf` | ملف التكوين المعياري الخاص بـ Debian |
| `~/.odoorc` | الملف الافتراضي الذي يُقرأ على Linux (تُحدَّد في `config.py:520`) |
| `~/.openerp_serverrc` | قديم بديل لـ `~/.odoorc` (إصدارات قديمة، مع تحذير Deprecation) |
| ملف بديل | يُحدَّد عبر الخيار `-c/--config` أو متغير البيئة `ODOO_RC` |

ملف التكوين بصيغة INI يحتوي على قسم `[options]` تليه خصائص (مثل `addons_path = ...`).

## 3. كيف يعمل التحميل أثناء الإقلاع

تسلسل الأولوية من الأدنى إلى الأعلى (يُطبَّق عبر `ChainMap` في `config.py:164`):

1. **القيم الافتراضية** `_default_options`
2. **ملف التكوين** `_file_options` — يُقرأ عبر `_load_file_options()` (في `config.py:901`)
3. **متغيرات البيئة** `_env_options` — مثل `ODOO_RC` وأي `ODOO_*` (في `config.py:617`). غالبية خيارات قاعدة البيانات لها مرادفات `PG*` (مثل `PGHOST`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`, `PGPORT`, `PGSSLMODE`)
4. **وسائط سطر الأوامر** `_cli_options` (في `config.py:627`)
5. **خيارات وقت التشغيل** `_runtime_options`

التهيئة تبدأ في `__init__` (في `config.py:185`):

```python
self.parser = self._build_cli()
self._load_default_options()
self._parse_config()
```

الاتصال الرئيسي `parse_config()` (في `config.py:553`) يُستدعى من `odoo/cli` عند بدء الإقلاع، ويقوم بـ:

- تحليل وسائط سطر الأوامر
- تحميل ملف التكوين (`_load_file_options`)
- إعداد الـ logging (`netsvc.init_logger()`)
- تهيئة مسار الوحدات (`modules.module.initialize_sys_path()`)

### معالجة المتغيرات المهمة أثناء التحميل (`_postprocess_options` — `config.py:649`)

- **`--init`/`--update`**: تحويل القوائم إلى ديكشيناري (مثل `{'base': True}` إذا كان `update=all`)
- **الوحدات العامة**: التأكد من إضافة الوحدات الإلزامية
- **`--dev=all`**: تحويله إلى `access`, `reload`, `qweb`, `xml`
- **خيارات الاختبار**: `--test-file` يُحوَّل إلى `test_tags` وإجباري `stop_after_init`

## 4. الخصائص الكاملة (جميع الخيارات)

### وحدات النظام والتطبيق

| الخيار | الوصف | الافتراضي |
| -------- | ------- | ----------- |
| `--load` / `server_wide_modules` | وحدات النظام العامة المُفعّلة عند الإقلاع | `base`, `rpc`, `web` |
| `-i` / `--init` | تثبيت وحدة أو أكثر (قائمة بفواصل، أو `all`) | `[]` |
| `-u` / `--update` | تحديث وحدة أو أكثر (يتطلب `-d`) | `[]` |
| `--reinit` | إعادة تهيئة وحدات (يتطلب `-d`) | `[]` |
| `--with-demo` | تثبيت البيانات التجريبية في قواعد بيانات جديدة | `False` |
| `--without-demo` | عدم تثبيت البيانات التجريبية (عكس السابق) | — |
| `--skip-auto-install` | تخطّي التثبيت التلقائي للوحدات `auto_install` | `False` |
| `-P` / `--import-partial` | للاستيراد الكبير مع تفادي فقدان التقدم | `''` |
| `--addons-path` | مسارات إضافية للإضافات (بفواصل) | `[]` |
| `--upgrade-path` | مسار ترقية إضافي | `[]` |
| `--pre-upgrade-scripts` | تشغيل نصوص قبل الترقية عند `-u` | `[]` |

```python
DEFAULT_SERVER_WIDE_MODULES = ['base', 'rpc', 'web']   # config.py:30
REQUIRED_SERVER_WIDE_MODULES = ['base', 'web']          # config.py:31
```

> الوحدات الإلزامية (`base`, `web`) تُضاف تلقائياً حتى لو لم تكن في القائمة.

### خدمة HTTP

| الخيار | الوصف | الافتراضي |
| -------- | ------- | ----------- |
| `--http-interface` | عنوان الواجهة للاستماع لبث HTTP | `0.0.0.0` |
| `-p` / `--http-port` | منفذ خدمة HTTP الرئيسية | `8069` |
| `--gevent-port` | منفذ عامل gevent | `8072` |
| `--no-http` | تعطيل خدمات HTTP و Longpolling تماماً | `False` |
| `--proxy-mode` | تفعيل وسائط Proxy (إعادة كتابة الهيدرز) | `False` |
| `--x-sendfile` | تفعيل X-Sendfile لتسليم الملفات عبر خادم الويب | `False` |

### واجهة الويب

| الخيار | الوصف | الافتراضي |
|--------|-------|-----------|
| `--db-filter` | تعبير Regex لتصفية قواعد البيانات لواجهة الويب | `''` |

### الاختبارات (Testing)

| الخيار | الوصف |
| -------- | ------- |
| `--test-file` | تشغيل ملف اختبار بايثون |
| `--test-enable` | تفعيل اختبارات الوحدات (يُلزم `--stop-after-init`) |
| `-t` / `--test-tags` | قائمة بفواصل لتصفية الاختبارات المنفذة |
| `--screencasts` | تخزين تسجيلات الشاشة في مجلد |
| `--screenshots` | تخزين لقطات الشاشة (الافتراضي `/tmp/odoo_tests`) |

### التسجيل (Logging)

| الخيار | الوصف | الافتراضي |
| -------- | ------- | ----------- |
| `--logfile` | ملف تخزين السجل | `''` |
| `--syslog` | إرسال السجل إلى خادم syslog | `False` |
| `--log-handler` | ضبط مستوى logger لوحدة معينة (قابل للتكرار) | `:INFO` |
| `--log-web` | اختصار لـ `--log-handler=odoo.http:DEBUG` | — |
| `--log-sql` | اختصار لـ `--log-handler=odoo.sql_db:DEBUG` | — |
| `--log-db` | قاعدة بيانات التسجيل | `''` |
| `--log-db-level` | مستوى تسجيل قاعدة البيانات | `warning` |
| `--log-config` | ملف تكوين JSON للتسجيل | `''` |
| `--log-level` | مستوى السجل (info, debug, warn, error, critical...) | `info` |

### البريد (SMTP)

| الخيار | الوصف | الافتراضي |
| -------- | ------- | ----------- |
| `--email-from` | عنوان المرسل للبريد | `''` |
| `--from-filter` | عنوان يُسمح له باستخدام SMTP | `''` |
| `--smtp` / `smtp_server` | خادم SMTP | `localhost` |
| `--smtp-port` | منفذ SMTP | `25` |
| `--smtp-ssl` | تشفير اتصالات SMTP بـ SSL (STARTTLS) | `False` |
| `--smtp-user` | اسم مستخدم SMTP | `''` |
| `--smtp-password` | كلمة مرور SMTP | `''` |
| `--smtp-ssl-certificate-filename` | شهادة SSL للمصادقة | `''` |
| `--smtp-ssl-private-key-filename` | المفتاح الخاص لشهادة SSL | `''` |

### قاعدة البيانات

| الخيار | الوصف | الافتراضي | بيئة |
| -------- | ------- | ----------- | ------ |
| `-d` / `--database` / `db_name` | قاعدة/قواعد البيانات عند تثبيت/تحديث الوحدات | `[]` | `PGDATABASE` |
| `-r` / `--db_user` | اسم مستخدم قاعدة البيانات | `''` | `PGUSER` |
| `-w` / `--db_password` | كلمة مرور قاعدة البيانات | `''` | `PGPASSWORD` |
| `--pg_path` | مسار ملف pg القابل للتنفيذ | `''` | `PGPATH` |
| `--db_host` | مضيف قاعدة البيانات | `''` | `PGHOST` |
| `--db_replica_host` | مضيف النسخة المتماثلة | `None` | `PGHOST_REPLICA` |
| `--db_port` | منفذ قاعدة البيانات | `None` | `PGPORT` |
| `--db_replica_port` | منفذ النسخة المتماثلة | `None` | `PGPORT_REPLICA` |
| `--db_sslmode` | وضع SSL للاتصال (disable/allow/prefer/require/verify-ca/verify-full) | `prefer` | `PGSSLMODE` |
| `--db_app_name` | اسم التطبيق في قاعدة البيانات (`{pid}` يُستبدل) | `odoo-{pid}` | `PGAPPNAME` |
| `--db_maxconn` | أقصى عدد اتصالات فيزيائية لـ PostgreSQL | `64` | — |
| `--db_maxconn_gevent` | أقصى اتصالات لعامل gevent تحديداً | `None` | — |
| `--db-template` | قالب قاعدة بيانات مخصص لإنشاء قاعدة جديدة | `template0` | `PGDATABASE_TEMPLATE` |
| `--db_replica_host` فارغ | يدعم وضع `replica` للتطوير | — | — |

### الترجمة (i18n)

| الخيار | الوصف |
|--------|-------|
| `--load-language` | اللغات التي تُحمَّل ترجماتها |
| `--i18n-overwrite` | الكتابة فوق مصطلحات الترجمة الموجودة عند تحديث وحدة |

### الأمان (Security)

| الخيار | الوصف | الافتراضي |
| -------- | ------- | ----------- |
| `--no-database-list` | تعطيل عرض قائمة قواعد البيانات ومدير/محدد قواعد البيانات | `True` |
| `admin_passwd` | كلمة المرور الفائقة (Super-admin) — تُخزَّن مشفّرة بـ `pbkdf2_sha512` | `admin` |
| `proxy_access_token` | رمز وصول للبروكسي | `''` |
| `publisher_warranty_url` | رابط الضمان | خدمات Odoo |

### الخيارات المتقدمة (Advanced)

| الخيار | الوصف | الافتراضي |
| -------- | ------- | ----------- |
| `--dev` / `dev_mode` | تفعيل ميزات المطور (access/qweb/reload/replica/werkzeug/xml، أو `all`) | `[]` |
| `--stop-after-init` | إيقاف السيرفر بعد التهيئة | `False` |
| `--osv-memory-count-limit` | حد أقصى للسجلات في جداول osv_memory الافتراضية | `0` (بدون حد) |
| `--transient-age-limit` | مدة الاحتفاظ بسجلات TransientModel (ساعات) | `1.0` |
| `--max-cron-threads` | عدد خيوط معالجة مهام cron المتزامنة | `2` |
| `--limit-time-worker-cron` | أقصى عمر لخيط/عامل cron قبل إعادة التشغيل | `0` |
| `--unaccent` | محاولة تفعيل امتداد unaccent عند إنشاء قواعد بيانات جديدة | `False` |
| `--geoip-city-db` | مسار ملف GeoIP City | `/usr/share/GeoIP/GeoLite2-City.mmdb` |
| `--geoip-country-db` | مسار ملف GeoIP Country | `/usr/share/GeoIP/GeoLite2-Country.mmdb` |

### تعدد المعالجات (Multiprocessing)

| الخيار | الوصف | الافتراضي |
| -------- | ------- | ----------- |
| `--workers` | عدد العمال، `0` يعطّل وضع prefork | `0` |
| `--limit-memory-soft` | أقصى ذاكرة افتراضية لكل عامل قبل إعادة التعيين | `2048MiB` |
| `--limit-memory-soft-gevent` | مثل السابق لكن لعامل gevent | مرتبط بـ soft |
| `--limit-memory-hard` | أقصى ذاكرة افتراضية، عندها يفشل أي تخصيص ذاكرة | `2560MiB` |
| `--limit-memory-hard-gevent` | مثل السابق لكن لعامل gevent | مرتبط بـ hard |
| `--limit-time-cpu` | أقصى وقت CPU لكل طلب | `60` ثانية |
| `--limit-time-real` | أقصى وقت حقيقي لكل طلب | `120` ثانية |
| `--limit-time-real-cron` | أقصى وقت حقيقي لكل مهمة cron | `-1` (يتبع `--limit-time-real`) |
| `--limit-request` | أقصى عدد طلبات لكل عامل | `65536` |

### خيارات الملف فقط (FileOnly — تُقرأ من ملف التكوين فقط، لا من CLI)

| الخيار | الوصف | الافتراضي |
| -------- | ------- | ----------- |
| `admin_passwd` | كلمة المرور الفائقة | `admin` |
| `bin_path` | مسار الملفات الثنائية | `''` |
| `csv_internal_sep` | فاصل CSV داخلي | `,` |
| `default_productivity_apps` | تفعيل تطبيقات الإنتاجية الافتراضية | `False` |
| `import_file_maxbytes` | أقصى حجم لملفات الاستيراد | `10MiB` |
| `import_file_timeout` | مهلة الاستيراد | `3` |
| `import_url_regex` | Regex لاستيراد الروابط | `^(http | https)://` |
| `proxy_access_token` | رمز الوصول للبروكسي | `''` |
| `publisher_warranty_url` | رابط الضمان | خدمات Odoo |
| `reportgz` | ضغط التقارير | `False` |
| `websocket_keep_alive_timeout` | مهلة إبقاء اتصال WebSocket حيّاً | `3600` |
| `websocket_rate_limit_burst` | حد معدل WebSocket (انفجار) | `10` |
| `websocket_rate_limit_delay` | تأخير معدل WebSocket | `0.2` |

## 5. الخصائص المحسوبة (Properties) والوظائف المساعدة

- **`root_path`** (`config.py:992`): جذر تثبيت Odoo.
- **`addons_base_dir`** (`config.py:996`): مسار إضافات الأساس (`root/addons`).
- **`addons_community_dir`** (`config.py:1000`): مسار إضافات المجتمع.
- **`addons_data_dir`** (`config.py:1003`): مجلد بيانات الإضافات (ضمن `data_dir`).
- **`session_dir`** (`config.py:1020`): مجلد الجلسات (ضمن `data_dir/sessions`).
- **`filestore(dbname)`** (`config.py:1031`): مجلد التخزين الملحقات لقاعدة بيانات.
- **`set_admin_password()` / `verify_admin_password()`** (`config.py:1034,1037`): تعيين/التحقق من كلمة المرور الفائقة (مع تشفير وتحديث التجزئة).
- **`http_socket_activation`** (`config.py:1050`): كشف تفعيل المقبس عبر systemd (`LISTEN_FDS`/`LISTEN_PID`).
- **`get(key, default)`** / **`__getitem__` / `__setitem__`**: الوصول للمتغيرات.
- **`save(keys)`** (`config.py:936`): حفظ الإعدادات إلى ملف التكوين (يُنشئ قسماً `[options]`).
- **`load()`** (قديم، `config.py:897`): تحميل خيارات الملف (Deprecated منذ 19.0).
- **`parse(name, value)`** (`config.py:864`): تحويل قيمة نصية لنوع الخيار.
- **`format(name, value)`** (`config.py:889`): تنسيق قيمة خيار لنص.
- **مدقّقات أنواع**: `_check_bool`, `_check_comma`, `_check_path`, `_check_addons_path`, `_check_upgrade_path`, `_check_scripts`, `_check_without_demo`.

## 6. ملاحظات إضافية

- **الأسماء المستعارة (Aliases)** (`config.py:179`): `import_image_maxbytes`→`import_file_maxbytes`, `import_image_regex`→`import_url_regex`, `import_image_timeout`→`import_file_timeout`.
- **أوضاع المطور** (`config.py:29`): `ALL_DEV_MODE = ['access', 'qweb', 'reload', 'xml']`، مع ميزات إضافية `replica` و`werkzeug`.
- **قاعدة البيانات المتماثلة**: وضع `replica` في `dev_mode` يُعادل الاستخدام القديم لـ `db_replica_host` الفارغ.
- **الأمان**: `admin_passwd` يُخزَّن مشفّراً عبر passlib بـ `pbkdf2_sha512` مع 600,000 جولة.
