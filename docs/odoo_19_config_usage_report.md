# خريطة استخدام خصائص التكوين في Odoo 19.0

> تقرير يوضّح **أين** و**كيف** و**متى** تُستدعى كل خاصية من خصائص `odoo/tools/config.py` داخل مشروع Odoo.
> المصدر: تحليل شامل لكامل الشيفرة في `odoo/` و `addons/`.

## 1. نقطة الدخول — متى يُحمَّل التكوين؟

| المكان | ماذا يحدث |
|--------|-----------|
| `odoo/cli/server.py:42` | أول فحص أثناء الإقلاع: إن كان مستخدم قاعدة البيانات `postgres` |
| `odoo/cli/server.py:54` | الطباعة عن ملف التكوين المُحمَّل (`config['config']`) |
| `odoo/cli/server.py:115` | قراءة `config['stop_after_init']` لتحديد متى يُوقف السيرفر |
| `odoo/tools/config.py:185-187` | `configmanager.__init__` يبني الـ CLI ويحمّل الخيارات تلقائياً |
| `odoo/tools/config.py:553` | `parse_config()` — يُستدعى أثناء بدء السيرفر لتجهيز كل الخيارات |

---

## 2. الخصائص المستخدمة في إقلاع الخادم (Server Boot)

### الملف الرئيسي: `odoo/service/server.py` و `odoo/cli/server.py`

| الخاصية | أين تُستخدم | كيف | متى |
|---------|------------|-----|-----|
| `http_interface` | `service/server.py:411` | تحديد واجهة الاستماع لـ HTTP | عند بناء `HttpServer` أثناء الإقلاع |
| `http_port` | `service/server.py:412` | منفذ الاستماع الرئيسي | عند بناء `HttpServer` |
| `gevent_port` | `service/server.py:770` | منفذ عامل gevent | عند بناء خادم WebSocket/gevent |
| `http_enable` | `service/server.py:653,1038,1081` | تفعيل/تعطيل خدمة HTTP | عند تشغيل الخادم وmind the loop |
| `workers` | `service/server.py:890,1626` | عدد العمال (prefork)؛ `0` = وضع خيطي | عند اختيار نموذج الخادم |
| `db_maxconn` | `service/server.py:258` | حساب `max_http_threads` | عند تهيئة الخادم |
| `max_cron_threads` | `service/server.py:622,1045` | عدد خيوط cron | عند تشغيل cron |
| `limit_request` | `service/server.py:892` | حد الطلبات لكل عامل | عند بناء عامل prefork |
| `limit_time_real` | `service/server.py:516,891` | أقصى وقت حقيقي للطلب/العامل | عند جدولة/إعادة تشغيل | 
| `limit_time_real_cron` | `service/server.py:518,893` | أقصى وقت لمهمة cron | عند تشغيل cron |
| `limit_time_cpu` | `service/server.py:1261,1295` | حد CPU للعامل (`setrlimit`) | عند معالجة الطلب |
| `limit_memory_soft` | `service/server.py:81,505,1285,1561` | إعادة تعيين العامل عند تجاوز الذاكرة | مراقبة مستمرة |
| `limit_memory_hard` | `service/server.py:81` | حد الذاكرة الصلب | مراقبة مستمرة |
| `limit_memory_soft_gevent` | `service/server.py:779` | حد ذاكرة عامل gevent | بناء عامل gevent |
| `limit_memory_hard_gevent` | `service/server.py:83-84` | حد ذاكرة عامل gevent | مراقبة مستمرة |
| `limit_time_worker_cron` | `service/server.py:564,612,1435` | عمر خيط cron قبل إعادة التشغيل | تشغيل cron |
| `server_wide_modules` | `service/server.py:1514` | تحميل وحدات النظام العامة | قبل تنظيط السجل/الوحدات |
| `init` / `update` / `reinit` | `service/server.py:1583-1585` | تمريرها إلى `Registry.new(...)` | عند فتح كل قاعدة بيانات |
| `test_enable` | `service/server.py:197,236,653,711,1588` | تفعيل وضع الاختبار (وقت أطول، عدم إيقاف) | عند تشغيل الاختبارات |
| `db_name` | `service/server.py:100` | تحديد قواعد البيانات أو سردها | عند بدء الخادم |
| `pidfile` | `cli/server.py:76-90` | كتابة/حذف معرّف العملية | بداية/نهاية الإقلاع |
| `db_host` / `db_user` / `db_port` | `cli/server.py:61-63` | تسجيل معلومات اتصال قاعدة البيانات | الإقلاع |
| `db_replica_host` / `db_replica_port` | `cli/server.py:65-70` | تفعيل وضع النسخة المتماثلة | الإقلاع |
| `upgrade_path` | `cli/server.py:57` | تسجيل مسار الترقية | الإقلاع |
| `pre_upgrade_scripts` | `cli/server.py:59`, `modules/loading.py:390` | تشغيل نصوص الترقية قبل تحميل الوحدات | عند `-u` |
| `stop_after_init` | `cli/server.py:115` | إيقاف السيرفر بعد التهيئة | نهاية التهيئة |

---

## 3. الخصائص المستخدمة في الاتصال بقاعدة البيانات

### `odoo/sql_db.py` و `odoo/tools/misc.py` و `odoo/service/db.py`

| الخاصية | أين تُستخدم | كيف | متى |
|---------|------------|-----|-----|
| `db_host` | `tools/misc.py:176` | `PGHOST` في بيئة أدوات pg | قبل تشغيل أدوات PostgreSQL |
| `db_port` | `tools/misc.py:178` | `PGPORT` | قبل تشغيل أدوات pg |
| `db_user` | `tools/misc.py:180` | `PGUSER` | قبل تشغيل أدوات pg |
| `db_password` | `tools/misc.py:182` | `PGPASSWORD` | قبل تشغيل أدوات pg |
| `db_app_name` | `tools/misc.py:184-185`, `sql_db.py:788` | `PGAPPNAME` + اسم تطبيق الاتصال | عند إنشاء اتصال DB |
| `db_sslmode` | `tools/misc.py:186` | `PGSSLMODE` | عند إنشاء اتصال DB |
| `pg_path` | `tools/misc.py:155-156` | موقع أدوات pg القابلة للتنفيذ | عند استدعاء pg |
| `bin_path` | `tools/misc.py:144-145` | إضافة مجلد ثنائيات إلى المسار | عند البحث عن pg |
| `db_maxconn` | `sql_db.py:823` | حجم مجمع اتصالات PostgreSQL | عند بناء `ConnectionPool` |
| `db_maxconn_gevent` | `sql_db.py:823` | مجمع اتصالات خاص بـ gevent | عند بناء pool لـ gevent |
| `db_template` | `sql_db.py:556`, `service/db.py:131,444` | قالب إنشاء قاعدة البيانات | عند إنشاء قاعدة بيانات جديدة |
| `db_name` | `sql_db.py:376`, `service/db.py:106,438,442` | فلترة/اختيار قواعد البيانات | عند فتح قواعد التأكد |
| `dbfilter` | `service/db.py:438-442` | تصفية قواعد البيانات المتاحة | عند سرد قواعد البيانات |
| `list_db` | `service/db.py:49,435,484` | إظهار/إخفاء قائمة قواعد البيانات | عند الوصول لمدير قواعد البيانات |
| `unaccent` | `service/db.py:155` | تفعيل امتداد unaccent | عند إنشاء قاعدة بيانات |
| `load_language` | `service/db.py:69`, `base/models/res_lang.py:253-259` | تحميل ترجمة لغة عند الإقلاع | عند تهيئة قاعدة البيانات |
| `admin_passwd` (via `verify_admin_password`) | `service/db.py:61`, `web/controllers/database.py:32-152` | التحقق من كلمة المرور الفائقة (Superadmin) | عند إدارة قواعد البيانات (إنشاء/نسخ/حذف) |
| `db_replica_host` | `sql_db.py:807-809`, `netsvc.py:131-136`, `orm/registry.py:258` | استخدام اتصال قراءة-فقط للنسخة المتماثلة | عند فتح الـ registry |

---

## 4. الخصائص المستخدمة في تهيئة التسجيل (Logging)

### `odoo/netsvc.py`

| الخاصية | خط | كيف | متى |
|---------|----|-----|-----|
| `log_config` | 234 | تحميل ملف dictconfig JSON | أثناء `init_logger()` في الإقلاع |
| `syslog` | 251 | توجيه السجل لنظام syslog | أثناء `init_logger()` |
| `logfile` | 261-263 | الكتابة في ملف سجل | أثناء `init_logger()` |
| `log_db` | 55, 294 | تسجيل لقاعدة بيانات | خلال تشغيل الخادم |
| `log_db_level` | 303 | مستوى تسجيل قاعدة البيانات | أثناء `init_logger()` |
| `log_level` | 307 | تعيين مستوى السجل عبر `PSEUDOCONFIG_MAPPER` | أثناء `init_logger()` |
| `log_handler` | 309 | قائمة معالجات السجل `MODULE:LEVEL` | أثناء `init_logger()` |

---

## 5. الخصائص المستخدمة في تحميل الوحدات

### `odoo/modules/module.py` و `odoo/modules/loading.py` و `odoo/modules/db.py`

| الخاصية | أين تُستخدم | كيف | متى |
|---------|------------|-----|-----|
| `addons_path` | `modules/module.py:149` | البحث في مسارات الإضافات عن الوحدات | بناء `ad_paths` للبحث عن الوحدات |
| `upgrade_path` | `modules/module.py:157` | مسار نصوص الترقية | عند البحث عن سكربتات الترقية |
| `server_wide_modules` | `base/models/ir_module.py:671`, `base/models/ir_http.py:388`, `base/models/ir_asset.py:315`, `web/controllers/webclient.py:30,59`, `website/models/ir_http.py:294`, `http.py:2761` | الوحدات العامة المثبّتة إلزامياً والمحمّلة في الراوتر والـ assets | بعد الإقلاع وأثناء معالجة HTTP |
| `skip_auto_install` | `modules/db.py:86`, `base/models/ir_module.py:431` | تخطي التثبيت التلقائي | عند تثبيت الوحدات الجديدة |
| `init` / `update` / `reinit` | `modules/loading.py:279` | تحديد وضع التثبيت/التحديث | عند تحميل الوحدات |
| `overwrite_existing_translations` | `modules/loading.py:233` | كتابة فوق الترجمات عند التحديث | أثناء تحميل الوحدات |
| `pre_upgrade_scripts` | `modules/loading.py:390` | تشغيل النصوص قبل الترقية | عند `-u` |
| `import_partial` | `base/models/ir_model.py:2649`, `tools/convert.py` | تفعيل الاستيراد المتدرّج | عند استيراد ملفات XML/فلي |

---

## 6. الخصائص المستخدمة في معالجة طلبات HTTP

### `odoo/http.py` و وحدات التحكم في `addons/web`

| الخاصية | أين تُستخدم | كيف | متى |
|---------|------------|-----|-----|
| `dbfilter` | `http.py:402-423` | تصفية قواعد البيانات حسب المضيف | عند كل طلب (نودب) |
| `db_name` | `http.py:420-423` | قائمة قواعد البيانات المتاحة | عند الطلبات بدون قاعدة بيانات |
| `data_dir` | `http.py:672`, `web/controllers/binary.py:58` | تحديد مسار `filestore` للمرفقات | عند تسليم المرفقات |
| `x_sendfile` | `http.py:670`, `web/controllers/binary.py:55` | تفويض تسليم الملفات للوكيل | عند تسليم ملفات كبيرة |
| `server_wide_modules` | `http.py:2761` | بناء الراوتر (نودب) بوحدات النظام | عند كل طلب غير مرتبط بقاعدة |
| `proxy_mode` | `http.py:2835` | إعادة كتابة هيدر `HTTP_X_FORWARDED_HOST` | عند كل طلب |
| `geoip_city_db` | `http.py:2785` | فتح قاعدة GeoIP City | عند دقة الموقع الجغرافي |
| `geoip_country_db` | `http.py:2796` | فتح قاعدة GeoIP Country | عند دقة الموقع الجغرافي |
| `dev_mode` (`werkzeug`) | `http.py:2348` | مفعّل منقّح HTML عند خطأ طلب | عند حدوث استثناء HTTP |
| `list_db` | `web/controllers/home.py:147`, `database.py:33` | صفحة قواعد البيانات / مديرها | عند الوصول لصفحة نودب |

---

## 7. الخصائص المستخدمة في ORM والـ Registry

### `odoo/orm/registry.py` و `odoo/orm/models_transient.py`

| الخاصية | أين تُستخدم | كيف | متى |
|---------|------------|-----|-----|
| `with_demo` | `registry.py:185` | تثبيت البيانات التجريبية | عند إنشاء قاعدة بيانات جديدة |
| `test_enable` | `registry.py:236` | تشغيل اختبارات ما بعد التثبيت | عند تحميل الـ registry |
| `db_replica_host` / `dev_mode`(`replica`) | `registry.py:258` | استخدام pool قراءة-فقط | عند فتح الـ registry |
| `osv_memory_count_limit` | `models_transient.py:24` | حد عدد سجلات TransientModel | عند استخدام الويزاردز |
| `transient_age_limit` | `models_transient.py:26` | مدة بقاء سجلات الويزارد | عند تنظيف الويزاردز |

---

## 8. خصائص وضع المطور (`dev_mode`) في الأساسيات

| الوضع | أين يُستخدم | كيف |
|-------|------------|-----|
| `reload` | `service/server.py:1657` | إعادة تشغيل السيرفر عند تغيّر الكود |
| `xml` | `ir_ui_view.py:225,3088`, `ir_qweb.py:999,2823`, `ir_rule.py:137`, `website_page.py:414,472`, `ir_asset.py:126` | قراءة العرض من الكود بدل قاعدة البيانات |
| `qweb` | `ir_qweb.py:1015,1328` | طباعة XML المحدّث عند أخطاء qweb |
| `access` | `http.py:2884` | طباعة traceback أخطاء الصلاحيات |
| `werkzeug` | `http.py:2348` | منقّح HTML لأخطاء الطلبات |
| `replica` | `registry.py:258`, `netsvc.py:131-136`, `cli/server.py:67` | محاكاة نسخة متماثلة للقراءة فقط |

---

## 9. خصائص البريد الإلكتروني (SMTP)

### `odoo/addons/base/models/ir_mail_server.py`

| الخاصية | خط | كيف | متى |
|---------|----|-----|-----|
| `smtp_server` | 480 | خادم SMTP الافتراضي | عند إرسال بريد دون خادم معيّن |
| `smtp_port` | 481 | منفذ SMTP (افتراضي 25) | عند إرسال بريد |
| `smtp_user` | 482 | مستخدم SMTP | عند المصادقة |
| `smtp_password` | 483 | كلمة مرور SMTP | عند المصادقة |
| `smtp_ssl` | 490 | تفعيل STARTTLS | عند إنشاء اتصال SMTP |
| `smtp_ssl_certificate_filename` | 492 | شهادة SSL للمصادقة | عند اتصال SMTP |
| `smtp_ssl_private_key_filename` | 493 | المفتاح الخاص لـ SSL | عند اتصال SMTP |
| `email_from` | 653,663 | عنوان المرسل الافتراضي (بديل `mail.default.from`) | عند تحديد مرسل الرسالة |
| `from_filter` | 676 | عنوان يُسمح له باستخدام إعداد SMTP (بديل `mail.default.from_filter`) | عند اختيار خادم SMTP |

> ملاحظة: `http_port` يُستخدم أيضاً في `ir_config_parameter.py:22` لتوليد URL الأساسي الافتراضي `web.base.url`.

---

## 10. خصائص الاختبار (Testing)

| الخاصية | أين تُستخدم | كيف | متى |
|---------|------------|-----|-----|
| `test_enable` | `tests/common.py`, `tests/shell.py`, `modules/loading.py`, `service/server.py`, `orm/registry.py` + ~20 ملفاً في `addons/` (مثل `account`, `l10n_*`, `point_of_sale`) | تفعيل/تخطّي عمليات حساسة للوقت (مزامنة فواتير، ملء بيانات افتراضية) | أثناء التشغيل بالاختبارات |
| `test_tags` | `tests/loader.py`, `tests/common.py`, `tests/shell.py`, `account/tests/common.py` | تصفية الاختبارات المنفذة | أثناء جمع/تشغيل الاختبارات |
| `test_file` | `addons/spreadsheet/models/spreadsheet_mixin.py` | مسار ملف اختبار فيروسات محدد | — |
| `screencasts` | `tests/common.py`, `base/tests/test_http_case.py` | مجلد تسجيلات الاختبارات | أثناء اختبارات HTTP |
| `screenshots` | `tests/common.py`, `base/tests/test_http_case.py` | مجلد لقطات الاختبارات | أثناء اختبارات HTTP |

---

## 11. خصائص في إضافات المجتمع (addons)

| الخاصية | الملف | كيف |
|---------|-------|-----|
| `websocket_rate_limit_burst` | `addons/bus/websocket.py:299` | حد انفجار معدل WebSocket |
| `websocket_rate_limit_delay` | `addons/bus/websocket.py:301` | تأخير معدل WebSocket |
| `websocket_keep_alive_timeout` | `addons/bus/websocket.py:827` | مهلة إبقاء اتصال WebSocket |
| `gevent_port` | `addons/bus/websocket.py:1028` | رسالة خطأ ربط المنفذ |
| `default_productivity_apps` | `addons/base_install_request/__init__.py:11` | تفعيل تطبيقات إنتاجية افتراضية عند الانضمام |
| `import_file_maxbytes` | `addons/base_import/models/base_import.py:1358`, `addons/mass_mailing/models/mailing.py:1492` | حد حجم الاستيراد من URL |
| `import_file_timeout` | `base_import.py:1361`, `mailing.py:1495` | مهلة تنزيل ملف الاستيراد |
| `import_url_regex` | `base_import.py:1294,1357,1555` | التحقق من صحة روابط الاستيراد |
| `publisher_warranty_url` | `addons/mail/models/update.py:71` | رابط ضمان الناشر |
| `proxy_access_token` | `addons/iot_drivers/cli/genproxytoken.py`, `iot_handlers/drivers/l10n_eg_drivers.py` | رمز وصول للبروكسي |
| `workers` | `base/models/ir_actions_report.py:119,141` | اختيار طريقة إنشاء التقارير (1 worker) |
| `limit_time_real` / `limit_time_real_cron` | `addons/cloud_storage_migration/models/ir_attachment.py` | مهلة الترحيل السحابي |
| `test_enable` | `addons/*` (spreadsheet, l10n_*, account, point_of_sale, hr_holidays...) | تخطّي عمليات خارجية/شبكات أثناء الاختبارات |

---

## 12. خصائص بدون استخدام فعلي في الشيفرة (تُدار خارجياً فقط)

| الخاصية | ملاحظة |
|---------|--------|
| `reportgz` | لا يُخشى استخدامها إلا في `base/tests/test_configmanager.py` — لا تستهلكها الشيفرة الحالية |
| `csv_internal_sep` | تُدار ضمن تنسيق الاستيراد/التصدير (غير مستهلكة مباشرة في هذه النسخة؛ تُستخدم في أدوات CSV قديمة) |
| `email_from` (كخاصية مباشرة) | تُستبدل القيمة الافتراضية عبر `ir.mail_server` و كذلك تُقرأ من `config.get("email_from")` في `ir_mail_server.py:653,663` |
| `admin_passwd` (المباشرة) | لا تُقرأ كنص أبداً؛ تُستخدم فقط عبر `verify_admin_password()` / `set_admin_password()` بسبب التشفير |
| `websocket_*` | متاحة للـ addons (bus) التي تستدعيها |
| `[config]` (مسار الملف) | يُستخدم في كل مكان `config['config']` (مثل `cli/server.py:54`) |

---

## 13. ملخص دورة حياة الاستدعاء "متى"

1. **التحميل الأول**: `cli/server.py` → `config.parse_config()` يجهّز كل الخيارات (ملف + بيئة + CLI).
2. **إقلاع الخادم**: `service/server.py` يقرأ `http_*`, `workers`, `limit_*`, `max_cron_threads`, `db_*`.
3. **تهيئة التسجيل**: `netsvc.init_logger()` يقرأ `log_*` و `syslog` و `logfile`.
4. **فك تشفير الوحدات**: `modules/module.py` يقرأ `addons_path`, `upgrade_path`; وحدات `server_wide_modules` تُحمَّل.
5. **فتح قاعدة البيانات**: `sql_db.py` + `misc.py` يبنيان بيئة `PG*` ويحدّدان مجمع الاتصالات.
6. **الـ Registry**: `orm/registry.py` يقرأ `with_demo`, `test_enable`, `replica`.
7. **معالجة الطلبات**: `http.py` يقرأ `dbfilter`, `proxy_mode`, `geoip_*`, `data_dir`, `x_sendfile`, `dev_mode`.
8. **وقت التشغيل**: cron/workers يستخدمون `limit_time_*`, `limit_memory_*`, و `ir.mail_server` يستخدم `smtp_*`.