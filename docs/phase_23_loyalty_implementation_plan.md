# خطة تنفيذ المرحلة 23 — الولاء والمكافآت (Loyalty & Rewards)

**الهدف**: بناء موديول Loyalty & Rewards متكامل بنمط Odoo 19.0 (`loyalty` / `sale_loyalty`) داخل مشروع cashflow_backend، يشمل إدارة البرامج والقواعد والمكافآت والبطاقات، مع تكامل كامل مع أمر البيع (كسب النقاط/الصرف كأسطر مكافأة + التسوية عند الاعتماد والإلغاء).

**المرجع**: `docs/advanced__infrastructured_implementation_plan.md` — المرحلة 23 (السطر 1237).

---

## القرارات المعتمدة

1. **تكامل كامل مع أمر البيع** بنمط Odoo: كسب النقاط من أسطر الأمر، صرف المكافآت كأسطر خصم/منتج (reward lines) داخل `sale_order_lines`، التسوية عند `confirm` وإعادة النقاط عند `cancel` عبر محرك نقاط نقي.
2. **الأنواع الثمانية للبرامج** مدعومة كاملة: `loyalty`, `coupons`, `promotion`, `promo_code`, `gift_card`, `ewallet`, `buy_x_get_y`, `next_order_coupons` — لكل منها قيم افتراضية مطابقة لـ `_program_type_default_values` في Odoo.
3. **`loyalty.mail`** يُخزَّن كتكوين فقط (trigger + points) دون إرسال بريد فعلي؛ لا توجد بنية `mail.template` في المشروع حالياً.

المداولات المسموحة: النقاط/المبالغ `float64` + تقريب 4 خانات عشرية + `NUMERIC(15,4)` (لا توجد مكتبة decimal في المشروع، القاعدة المعتمدة منذ المراحل السابقة).

---

## ملفات الترتيب والتنفيذ

```
docs/phase_23_loyalty_implementation_plan.md      (هذا الملف)
migrations/000040_create_loyalty_schema.{up,down}.sql
migrations/000041_seed_loyalty_acl.{up,down}.sql
internal/domain/loyalty/loyalty.go                 (الكيانات + Validate + ApplyProgramTypeDefaults)
internal/domain/loyalty/ports.go                   (واجهة Repository)
internal/domain/loyalty/points.go                  (محرك النقاط النقي: ComputeProgramPoints + DiscountAmount)
internal/usecase/loyalty/usecase.go               (CRUD البرامج/القواعد/المكافآت/البريد/البطاقات + generate/balance)
internal/usecase/loyalty/redeem.go                (ApplyCode / Claims / Earn / Redeem + Expressions المساعدة)
internal/usecase/loyalty/sale_integration.go      (SettleOrder / ReverseOrder)
internal/adapters/http/loyalty/dto.go
internal/adapters/http/loyalty/handler.go
internal/adapters/http/loyalty/routes.go
internal/adapters/storage/loyalty/postgres_repo.go
internal/adapters/storage/loyalty/memory_repo.go
```

### ملفات معدّلة

```
internal/domain/sale/order.go                      (إضافة حقول المكافأة على SaleOrderLine + حقلَي البطاقات/الكود على SaleOrder)
internal/adapters/storage/sale/postgres_repo.go    (حفظ/قراءة الأعمدة الجديدة)
internal/usecase/sale/sale_usecase.go              (LoyaltyService اختياري + SettleOnConfirm / ReverseOnCancel)
internal/usecase/usercases.go                      (عضو Loyalty + الحقن)
internal/adapters/http/handlers.go                 (عضو Loyalty handler)
internal/adapters/http/router.go                   (تثبيت /api/v1/loyalty)
internal/adapters/storage/repositories.go          (عضو Loyalty repo + المحدّثات)
docs/advanced__infrastructured_implementation_plan.md  (تحديث حالة المرحلة 23 بعد التنفيذ)
```

---

## قاعدة البيانات

### `000040_create_loyalty_schema.up.sql` — 7 جداول + أعمدة دمج

| الجدول | الغرض |
|---|---|
| `loyalty_programs` | البرنامج (الاسم، النوع، applies_on، trigger، الحدود، التاريخ، visibility، sale_ok) |
| `loyalty_program_pricelists` | علاقة برنامج ↔ قائمة أسعار |
| `loyalty_rules` | القاعدة (نقاط المكافأة، mode، الحد الأدنى، المنتجات/الفئات، code) |
| `loyalty_rewards` | المكافأة (خصم %/ثابت/لكل نقطة، product، required_points، discount_line_product_id) |
| `loyalty_cards` | البطاقة/الكوبون (كود فريد، نقاط، شريك، order_id، use_count) |
| `loyalty_card_history` | سجل الرصيد (issued / used بالربط بأمر) |
| `loyalty_mails` | التكوين فقط (create / points_reach) |
| `sale_order_coupon_points` | نقاط البرنامج المتوقعة لكل أمر+بطاقة (UNIQUE(order_id, coupon_id)) |

تعديلات دمج:

```sql
ALTER TABLE sale_orders
    ADD COLUMN IF NOT EXISTS applied_coupon_ids BIGINT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS code_enabled_rule_ids BIGINT[] DEFAULT '{}';

ALTER TABLE sale_order_lines
    ADD COLUMN IF NOT EXISTS reward_id BIGINT,
    ADD COLUMN IF NOT EXISTS coupon_id BIGINT,
    ADD COLUMN IF NOT EXISTS reward_identifier_code VARCHAR(64),
    ADD COLUMN IF NOT EXISTS points_cost NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    ADD COLUMN IF NOT EXISTS is_reward_line BOOLEAN NOT NULL DEFAULT false;
-- + FK reward_id → loyalty_rewards(id), coupon_id → loyalty_cards(id) (RESTRICT)
```

### `000041_seed_loyalty_acl.up.sql`

نفس نمط `000031/000039`: `res_group_permissions` مع `ON CONFLICT (group_id, model) DO NOTHING` للموديلات: `loyalty.program`, `loyalty.rule`, `loyalty.reward`, `loyalty.card`, `loyalty.card.history`, `loyalty.mail`, `sale.order.coupon.points`. (مجموعة 1: قراءة/إنشاء/تحديث بلا حذف؛ مجموعة 2: كاملة).

---

## منطق النقاط (مطابق لـ Odoo `_program_check_compute_points`)

لكل برنامج والمرشحين الأوتوماتيك أو المفعّلين بالكود:

- **المنتجات المؤهلة**: product_ids ∨ (الفئة + الأولاد) ∨ product_tag_id؛ إن كانت كلها فارغة: كل المنتجات (عدا برامج gift_card/ewallet لأغراض الكسب تُترك ضمن الحدود).
- **الحد الأدنى للكمية**: مجموع كميات الأسطر المطابقة ≥ `minimum_qty`.
- **الحد الأدنى للمبلغ**: مبلغ الأسطر المطابقة (شامل/صافي الضريبة حسب `minimum_amount_tax_mode`) ≥ `minimum_amount`.
- **نقاط كل قاعدة حسب `reward_point_mode`**:
  - `order`: `reward_point_amount` لنقطة واحدة (عند استيفاء الشروط).
  - `money`: `floor(reward_point_amount × المبلغ المدفوع)` حيث المبلغ = مجموع `price_total` للأسطر المطابقة (مع استبعاد أسطر مكافآت برامج ewallet/gift_card/البرنامج الحالي).
  - `unit`: `reward_point_amount × الكمية المطابقة`.
- أسطر المكافأة نفسها مستبعدة دائماً من الحساب.
- للمكافآت: `DiscountAmount(availablePoints, pointsToUse, eligibleAmount)` حسب `discount_mode`:
  - `percent`: `eligibleAmount × discount%`، التكلفة = `required_points`.
  - `per_order`: مبلغ ثابت `discount`، التكلفة = `required_points`.
  - `per_point`: `pointsToUse × discount` (تتطلب `pointsToUse ≥ required_points` وكامل الرصيد عند عدم التحديد)، مع سقف `discount_max_amount`.

`availablePoints` لكل بطاقة (نمط `_get_real_points_for_coupon`):
`card.points + (نقاط الأمر للكوبون إن كان البرنامج applies_on≠future وقبل الاعتماد) − نقاط أسطر المكافأة المستخدمة`.

---

## تكامل أمر البيع

- `sale.LoyaltyService` (تُعرّف داخل `saleusecase` لتجنب الدورات):
  - `SettleOrder(ctx, *sale.SaleOrder)`: بعد الاعتماد — إضافة النقاط المكتسبة للبطاقات، خصم `points_cost` من كل بطاقة، تحديث `use_count`، كتابة `loyalty_card_history`.
  - `ReverseOrder(ctx, *sale.SaleOrder)`: عند الإلغاء — إرجاع `points_cost`، سحب النقاط المكتسبة، مسح `sale_order_coupon_points`.
- `ConfirmOrder` (السطر 306) و`CancelOrder` (السطر 351) يستدعيان الخدمة عند وجودها (optional dep بنمط `NewSaleStockUseCase`).

---

## واجهات API (مثبتة تحت `/api/v1/loyalty`)

```
GET/POST   /programs                     قائمة / إنشاء برنامج (+ defaults حسب النوع)
GET/PUT/DELETE /programs/{id}            عرض / تحديث / تعطيل
PUT        /programs/{id}/type           تغيير النوع (يعيد توليد defaults)
POST/PUT/DELETE /programs/{id}/rules     إدارة القواعد
POST/PUT/DELETE /programs/{id}/rewards   إدارة المكافآت (ينشئ منتج خط الخصم تلقائياً)
POST/PUT/DELETE /programs/{id}/mails     إدارة البريد (تكوين فقط)
GET        /cards?program_id=&partner_id=  قائمة البطاقات (ترقيم صفحات)
GET        /cards/{id}                  بطاقة
POST       /cards/generate              توليد بطاقات {program_id, partner_id, count}
POST       /cards/{id}/balance          تعديل الرصيد {delta, description}
POST       /cards/{id}/archive          أرشفة
GET        /cards/{id}/history          سجل الرصيد
GET        /orders/{orderID}/claims     المكافآت القابلة للصرف
POST       /orders/{orderID}/earn       حساب/تثبيت النقاط للبرامج الأوتوماتيك
POST       /orders/{orderID}/redeem     صرف مكافأة {reward_id, coupon_id?/code?, points?}
POST       /orders/{orderID}/apply-code تفعيل كود {code}
GET        /check/{code}                فحص كود بطاقة/كود برومو
```

---

## الاختبارات والتحقق

- `go build ./...`, `go vet ./...`, `go test ./...`.
- وحدة محرك النقاط: حالات `order/money/unit` + الحدود + الحسم (في `internal/domain/loyalty`).
- اختبار `migrations` اللفاف + سلامة الجداول الجديدة.

---

## ملاحظات تبسيط معتمدة

- `product_tag` غير موديلي (لا يوجد `product.tag` في المشروع) → يُخزَّن `product_tag_id` لكن لا يُقيَّم في تطابق المنتجات.
- تقييم `product_domain` النصي غير مدعوم (يُخزَّن فقط؛ المطابقة عبر الأعمدة المنظمة).
- `reward_point_split` يدعم mode money/unit لكن يُجمَّع النقاط على بطاقة واحدة بدل عدة كوبونات.
- بطاقات برامج `future` تُنشأ وتُرتبط بأمر المصدر (`order_id`).
- `GET /partners/{id}/loyalty-cards` يُستفتى عبر `GET /loyalty/cards?partner_id=` (لا تظليل لموجه partners).

---

## حالة التنفيذ

| الخطوة | الحالة |
|---|---|
| كتابة خطة التنفيذ في docs | ✅ |
| Migrations 000040 + 000041 | ✅ |
| Domain (loyalty.go + ports.go + points.go) | ✅ |
| Storage (postgres + memory) | ✅ |
| تعديل sale domain + repos | ✅ |
| Usecase (usecase.go + redeem.go + sale_integration.go) | ✅ |
| HTTP (dto + handler + routes) | ✅ |
| الربط (repositories/usercases/handlers/router/sale.New) | ✅ |
| اختبارات + build/vet + تحديث حالة المرحلة 23 | ✅ |