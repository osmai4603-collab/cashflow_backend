# تحليل تحسين معمارية قاعدة البيانات واختصار الجداول

يستعرض هذا المستند تحليلاً معمارياً تفصيلياً لملف `init_database.sql` بهدف تبسيط البنية، تقليل عدد الجداول، وتحديد التغييرات على مستوى الحقول.

---

## 1. تبسيط نظام الوسوم (Tags Optimization)

**الهدف:** التخلص من جداول الربط واستخدام مصفوفات PostgreSQL.

### جداول سيتم حذفها

* `crm_tags`, `crm_lead_tags`
* `project_task_tags`, `project_task_tag_rel`
* `fleet_vehicle_tags`, `fleet_vehicle_tag_rel`

### التغييرات في الحقول

* **جدول `crm_leads`:**
  * **إضافة:** `tag_ids BIGINT[]` (تخزين معرفات الوسوم كمصفوفة).
* **جدول `project_tasks`:**
  * **إضافة:** `tag_ids BIGINT[]`.
* **جدول `fleet_vehicles`:**
  * **إضافة:** `tag_ids BIGINT[]`.

---

## 2. توحيد بنية المنتجات (Product Schema Consolidation)

**الهدف:** دمج القوالب والمتغيرات في كيان واحد لتبسيط منطق العمل.

### جداول سيتم حذفها

* `product_variants`
* `product_variant_attributes`

### التغييرات في الحقول

* **جدول `product_templates`:**
  * **إضافة (منقول من Variants):** `sku` (VARCHAR), `barcode` (VARCHAR), `extra_price` (NUMERIC).
  * **تعديل:** تحديث جميع الجداول التي تشير إلى `variant_id` لتشير مباشرة إلى `template_id`.

---

## 3. دمج صلاحيات الفرق (RBAC Simplification)

**الهدف:** توحيد إدارة المجموعات والفرق في جدول واحد.

### جداول سيتم حذفها

* `maintenance_teams`
* `maintenance_team_members`

### التغييرات في الحقول

* **جدول `res_groups`:**
  * **إضافة:** `group_type` (VARCHAR - لتمييز فرق الصيانة عن مجموعات الأمان)، `color` (INT)، `company_id` (BIGINT).
* **جدول `maintenance_equipment`:**
  * **تعديل:** تغيير مرجع `team_id` ليشير إلى `res_groups(id)`.
* **جدول `maintenance_requests`:**
  * **تعديل:** تغيير مرجع `team_id` ليشير إلى `res_groups(id)`.

---

## 4. تحسين نظام العناوين (Geopolitical Integration)

**الهدف:** الانتقال من النصوص الحرة إلى البيانات المهيكلة.

### التغييرات في الحقول

* **جداول `res_partners` و `res_companies`:**
  * **حذف (نصوص):** `city`, `state`, `country`.
  * **إضافة (روابط):** `country_id` (BIGINT REFERENCES res_country), `state_id` (BIGINT REFERENCES res_country_state).
  * **ملاحظة:** سيتم الإبقاء على `street` و `street2` و `zip_code` كحقول نصية لمرونتها.

---

## 5. تبسيط روابط المبيعات والمشتريات

**الهدف:** تقليل جداول الربط التي لا تدعم علاقات (متعدد لمتعدد) حقيقية.

### جداول سيتم حذفها

* `sale_order_invoices`
* `purchase_order_invoices`

### التغييرات في الحقول

* **جدول `account_moves` (الفواتير):**
  * **إضافة:** `source_order_id` (BIGINT - لربط الفاتورة بطلب المبيعات أو المشتريات الأصلي مباشرة).

---

## الخلاصة والتوصيات

هذه التغييرات ستؤدي إلى حذف **12 جدولاً** بشكل نهائي.

| الجدول المتأثر | الحقول المضافة | الحقول المحذوفة | المراجع المحدثة |
| :--- | :--- | :--- | :--- |
| `res_partners` | `country_id`, `state_id` | `city`, `state`, `country` | `res_country`, `res_country_state` |
| `product_templates` | `sku`, `barcode`, `extra_price` | - | - |
| `res_groups` | `group_type`, `color` | - | - |
| `crm_leads` | `tag_ids` | - | - |
| `account_moves` | `source_order_id` | - | `sale_orders` / `purchase_orders` |

---
**تاريخ التحليل:** ٢٢ مايو ٢٠٢٤
**الحالة:** مكتمل تقنياً وجاهز للتنفيذ.
