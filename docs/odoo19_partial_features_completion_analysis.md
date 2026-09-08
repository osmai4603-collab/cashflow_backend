# تحليل وإكمال الميزات الموجودة جزئياً مقارنة بسلوك Odoo 19

**تاريخ التحليل:** 2026-09-08  
**مرجع Odoo:** `/home/osm/Downloads/odoo-19.0/odoo-19.0`  
**المشروع المستهدف:** `cashflow_backend`  
**نوع الوثيقة:** تحليل فجوات ومواصفة سلوك قبل التنفيذ

## 1. الهدف

هذا التقرير لا يعيد جرد الميزات الغائبة بالكامل، بل يركز على الميزات التي لها كود جزئي في المشروع أو تعتمد على مجال موجود، ثم يحدد ما يلزم لإكمال سلوكها بما يقترب من Odoo 19 بشكل صارم.

المقصود بـ «بشكل صارم» هو مطابقة **السلوك التجاري والتكاملات وحالات الانتقال والقيود والآثار الجانبية**، وليس نسخ أسماء نماذج Odoo أو واجهاته حرفياً. لا يكفي أن يوجد جدول أو endpoint يحمل اسماً مشابهاً؛ يجب أن تكون دورة العمل متسقة، ذرية، قابلة للتدقيق، ومحمية من التكرار.

هذه الوثيقة تحليل ومواصفة تنفيذ. لا تعني أن البنود المذكورة قد نُفذت بالفعل.

## 2. طريقة التقييم

تم استخدام المستويات التالية:

| المستوى | المعنى |
|---|---|
| **موجود جزئياً** | توجد كيانات ومنطق أو routes، لكن جزءاً مهماً من دورة Odoo أو تكاملاتها مفقود. |
| **موجود شكلياً** | يوجد حقل أو جدول أو قيمة، لكن لا يوجد السلوك الذي يجعل الميزة عملية. |
| **متناقض** | التنفيذ الحالي يعطي نتيجة تختلف عن عقد Odoo أو يسمح بانتقالات أو حالات لا يسمح بها Odoo. |
| **مفقود** | لا توجد طبقة تشغيلية كافية، ويجب إنشاء مجال أو خدمة جديدة. |
| **شرط اكتمال** | سلوك قابل للاختبار يثبت أن الفجوة أغلقت. |

### مصادر المشروع التي تمت مراجعتها

- [internal/domain](../internal/domain)
- [internal/usecase](../internal/usecase)
- [internal/adapters/storage](../internal/adapters/storage)
- [internal/adapters/http](../internal/adapters/http)
- [migrations](../migrations)
- [router.go](../internal/adapters/http/router.go)

### مصادر Odoo الأساسية

- `addons/mail/models/mail_thread.py`
- `addons/mail/models/mail_followers.py`
- `addons/mail/models/mail_template.py`
- `addons/payment/models/payment_provider.py`
- `addons/payment/models/payment_transaction.py`
- `addons/account/models/account_move.py`
- `addons/account_edi/models/account_edi_document.py`
- `addons/stock/models/stock_move.py`
- `addons/stock/models/stock_move_line.py`
- `addons/stock/models/stock_rule.py`
- `addons/stock/models/stock_scrap.py`
- `addons/stock/models/stock_package.py`
- `addons/mrp/models/mrp_production.py`
- `addons/mrp/models/stock_rule.py`
- `addons/sale/models/sale_order.py`
- `addons/purchase/models/purchase_order.py`
- `addons/project/models/project_task.py`
- `addons/crm/models/crm_lead.py`
- `addons/hr/models/hr_employee.py`
- `addons/loyalty/models/loyalty_program.py`

## 3. الخلاصة التنفيذية

### أعلى فجوات يجب إغلاقها أولاً

1. **Mail/Chatter/followers:** لأن كل المجالات تعتمد عليها، ووجود `mail_messages` و`mail_activities` لا ينشئ Chatter تلقائياً.
2. **التكاملات العابرة بين Sales/Purchase/Stock/Accounting/MRP:** لأن الطلبات حالياً تحمل روابط وعدادات، لكن يجب أن تكون هذه القيم ناتجة عن الحركات والفواتير الفعلية.
3. **Payment Provider وEDI:** لأن الحالة الحالية تمثل دفعات داخلية، بينما بوابات الدفع والفوترة الإلكترونية تحتاج state machines وأماناً وتكراراً آمناً.
4. **Stock routes/packaging/scrap/traceability:** لأن `route_id` و`package_id` وlot لا تكفي وحدها لتنفيذ سلوك المخزون في Odoo.
5. **إزالة السلوك التجريبي والتناقضات:** خصوصاً ZATCA، ومولد XML، والتحقق من حالات الانتقال، وتزامن Memory/PostgreSQL.

## 4. Mail وChatter وFollowers وMail Templates

### 4.1 السلوك المرجعي في Odoo

`mail.thread` طبقة عامة يمكن أن تستخدمها نماذج الأعمال. عند إنشاء أو تعديل سجل قابل للمراسلة، يستطيع النظام:

- إنشاء رسالة داخل thread السجل عبر `message_post`.
- تحديد المتابعين عبر `message_subscribe` و`message_unsubscribe`.
- إرسال الرسالة حسب subtype والمستخدمين المتابعين.
- تسجيل تغيّر الحقول بقيمة قديمة وقيمة جديدة.
- ربط attachments وmentions والردود بالرسالة.
- إنشاء activities من القوالب أو من إجراءات السجل.
- تطبيق صلاحية قراءة thread حسب السجل الأصلي والبوابة.

الرسالة ليست مجرد صف يحتوي `res_model` و`res_id`. هي جزء من دورة نشر لها author، subtype، recipients، parent message، attachments، notification status، وaccess rules.

### 4.2 الموجود في المشروع

المشروع يملك:

- `activity.Activity` مع `ResModel` و`ResID` وموعد وإسناد.
- رسائل وإشعارات وطابور بريد في [internal/domain/activity/message.go](../internal/domain/activity/message.go).
- حالات طابور بريد وعامل SMTP في [internal/infrastructure/worker/email_worker.go](../internal/infrastructure/worker/email_worker.go).
- بث إشعارات لحظية عبر notification bus.

### 4.3 الناقص

| العنصر | الفجوة |
|---|---|
| `mail.thread` | لا توجد واجهة عامة تستعملها جميع النماذج القابلة للمراسلة. |
| Followers | لا توجد علاقة عامة `res_model/res_id/partner_id/subtypes`. |
| Subtypes | لا توجد أنواع أحداث مثل creation، stage change، assignment، note، أو email. |
| Field tracking | لا يوجد سجل ذري للقيمة القديمة والجديدة أثناء update. |
| Composer | لا توجد خدمة تنشئ رسالة وتحدد recipients وattachments وreply-to. |
| Templates | لا توجد قوالب ذات variables وrendering وصلاحيات واختبار preview. |
| Mentions | لا يوجد تحليل `@user` وربطه بمستخدم وإشعار آمن. |
| Portal/thread access | لا يوجد عرض thread للعميل بحسب صلاحية السجل. |
| Retry semantics | طابور البريد لا يمثل حالة notification لكل مستلم في رسالة متعددة المستلمين. |

### 4.4 التناقضات الحالية

1. وجود `ResModel` و`ResID` اختياريين يسمح بإنشاء رسالة مرتبطة بسجل دون التحقق من وجود السجل أو صلاحية الوصول إليه.
2. النشاط الحالي موجه إلى `AssignedUserID`، بينما Odoo يستطيع إيصال النشاط والرسالة إلى متابعين أو شركاء بحسب subtype.
3. لا توجد معاملة واحدة تضمن حفظ تعديل السجل ورسالة تتبع التعديل معاً.
4. `mail_messages` قد تُفهم كبديل لـ Chatter، لكنها لا تمثل thread أو followers أو field tracking.

### 4.5 التنفيذ المطلوب

1. إنشاء `mail.ThreadService` بعمليات: `PostMessage`, `Subscribe`, `Unsubscribe`, `ListFollowers`, `ListThread`.
2. إنشاء جداول مستقلة للرسائل، subtypes، followers، message recipients، tracking values، والمرفقات.
3. تعريف قائمة models قابلة للمراسلة عبر contract واضح، لا عبر قبول أي string.
4. ربط Sale/Purchase/Stock/CRM/Project/HR/Maintenance بالتسجيل التلقائي للأحداث المهمة.
5. إضافة renderer آمن للقوالب يمنع تنفيذ كود أو الوصول إلى حقول غير مسموحة.
6. جعل الإشعارات idempotent لكل `message_id + recipient_id + channel`.
7. إضافة صلاحيات thread والمرفقات والمتابعين، مع احترام company وrecord rules.

### 4.6 شروط القبول

- تعديل مرحلة CRM ينشئ tracking value ورسالة subtype صحيحة داخل نفس المعاملة.
- إضافة متابع لا ترسل له رسائل تخص subtype غير مشترك فيه.
- فشل SMTP لا يعيد إنشاء الرسالة ولا يرسل duplicate notification.
- مستخدم لا يملك قراءة السجل لا يستطيع قراءة thread حتى لو عرف `message_id`.
- حذف أو أرشفة السجل لا يؤدي إلى orphan messages غير قابلة للتدقيق.

## 5. Mass Mailing وMarketing Automation وSMS وLive Chat

### 5.1 الوضع الحالي

هذه الميزات لا تملك في المشروع مجالاً كاملاً. عامل البريد الحالي يعالج queue لرسائل بريد، لكنه لا يعادل حملات Odoo أو automation engine أو SMS أو Live Chat.

### 5.2 Mass Mailing

#### وظيفة الميزة

حملة بريد جماعية تتكون من قائمة مستلمين، رسالة وقالب، segment، حالة اشتراك، جدولة، وحالات تسليم وفتح ونقر وارتداد وإلغاء اشتراك.

#### النقص

- mailing campaigns وmailing lists وcontacts.
- consent وunsubscribe وbounce suppression.
- recipient state مستقل لكل حملة.
- tracking links وopen/click events.
- جدولة وحجم دفعة ومعدل إرسال.
- منع إرسال الحملة نفسها مرتين عند إعادة تشغيل worker.

#### شرط الإكمال

يجب أن تكون الحملة قابلة للاستئناف، وأن يكون لكل recipient مفتاح idempotency وحالة مستقلة، وأن يمنع النظام الإرسال لمن ألغى اشتراكه.

### 5.3 Marketing Automation

#### وظيفة الميزة

محرك يراقب حدثاً أو شرطاً على سجل، ثم ينفذ أنشطة بعد تأخير أو عند تحقق شرط، مثل إرسال بريد أو إنشاء activity أو تغيير مرحلة lead.

#### النقص والتناقض

- لا يوجد trigger/event registry.
- لا يوجد سجل execution لكل target.
- لا يوجد delay scheduling أو cancellation عند تغير الشرط.
- لا توجد حماية من loop عندما يغير automation السجل الذي أطلقه.

#### شرط الإكمال

كل تنفيذ يجب أن يملك `campaign_id`, `target_id`, `activity_id`, `scheduled_at`, `state`, و`attempt_count`، مع قيد يمنع تنفيذ نفس النشاط مرتين.

### 5.4 SMS

#### وظيفة الميزة

رسالة قصيرة مرتبطة بعميل أو سجل أعمال، تمر عبر مزود خارجي مع حالة queued/sent/delivered/failed، وإلغاء اشتراك وإعادة محاولة.

#### النقص

- provider abstraction وcredentials.
- أرقام موحدة والتحقق من البلد.
- opt-out وconsent.
- delivery callbacks وتوقيع webhook.
- حدود التكلفة والرصيد وrate limiting.

### 5.5 Live Chat

#### وظيفة الميزة

قناة محادثة فورية تربط زائراً أو عميلاً بمشغل أو فريق، مع جلسة، presence، توجيه، transcript، ومرفقات.

#### النقص

- channel/operator/team routing.
- guest identity وsession lifecycle.
- WebSocket أو SSE مع authentication مناسب.
- transcript مرتبط بـ partner أو lead عند التعرف عليه.
- presence وtyping وattachments وretention.

## 6. Accounting وPayment Providers وEDI

### 6.1 Accounting

### السلوك المرجعي

Odoo يفصل بين المسودة والاعتماد والترحيل، ويجعل الفاتورة قيداً محاسبياً له خطوط وضرائب وحسابات وتسويات وأثر على الرصيد. تعديلات المستند المرحل لا تكون تعديلاً صامتاً؛ بل تتم عبر عكس أو إشعار دائن أو آلية تصحيح مناسبة.

### الموجود

- حسابات ودفاتر وضرائب وحركات.
- مدفوعات داخلية وتسويات وكشوف بنكية.
- تقارير وحساب تحليلي.

### الناقص أو غير المكتمل

- منع التعديل غير المراقب على move مرحل.
- قواعد أقوى لتوازن المدين والدائن والعملة والفترة.
- تسوية متعددة السطور مع partial/full matching وexchange differences.
- fiscal positions وtax repartition وcash basis semantics.
- payment methods وoutstanding accounts وbank synchronization.
- قوالب تقارير ديناميكية كاملة وتوطينات الدول.

### خطر التناقض

اسم `Payment` في المشروع يمثل `account.payment` الداخلي، لكنه لا يمثل `payment.transaction` الخارجي. يجب عدم خلط الحالتين في API أو قاعدة البيانات.

### 6.2 Payment Providers

### وظيفة الميزة

مزود الدفع الخارجي يعالج طلب دفع أو authorization أو capture أو refund أو void، ثم يرسل callback قد يصل أكثر من مرة أو يصل بترتيب مختلف.

### الموجود

المشروع يملك دفعات inbound/outbound وحالات draft/posted/reconciled/cancelled في [internal/domain/payment/payment.go](../internal/domain/payment/payment.go). هذا مفيد للمحاسبة، لكنه لا يحتوي transaction/provider/token ecosystem.

### النقص الصريح

- provider وmethod وtoken.
- transaction state machine.
- payment links وreturn URLs.
- webhook signature validation.
- idempotency key وprovider reference.
- refund/capture/void.
- حفظ request/response دون تسريب الأسرار.

### شرط الإكمال

يجب رفض أي انتقال غير مسموح، ومعالجة callback المكرر دون أثر ثانٍ، وربط transaction بالفاتورة أو أمر البيع، ثم إنشاء payment داخلي مرة واحدة فقط بعد نجاح العملية.

### 6.3 EDI وZATCA

### الموجود

توجد كيانات EDI وصيغ ZATCA/UBL/Peppol، لكن [zatca_processor.go](../internal/usecase/accounting/zatca_processor.go) يحتوي TODOs، و`GenerateQRCode` يعيد TLV تجريبياً، و`ValidateXML` و`SignXML` غير منفذين فعلياً.

### النقص والتناقض

1. XML الحالي ليس مولداً كاملاً لكل خطوط الفاتورة والضرائب والأطراف والإجماليات.
2. التحقق يعيد نجاحاً دون XSD/Schematron حقيقي.
3. التوقيع يعيد XML كما هو بدلاً من canonicalization وتوقيع XAdES.
4. QR code يستخدم قيمة اختبار ثابتة بدلاً من بيانات البائع والفاتورة.
5. لا توجد دورة إرسال خارجية موثقة مع retry وresponse validation وstatus history.
6. تخزين `PrivateKey` في نموذج عادي يحتاج حماية تشفيرية وإدارة أسرار.

### شرط الإكمال

- XML يمرر schema وSchematron لكل نوع وثيقة.
- QR/TLV يحسب القيم الفعلية ويرفض الحقول الناقصة.
- التوقيع يتحقق بمفتاح الشهادة المقابلة.
- كل محاولة إرسال لها request id وresponse وhash وحالة نهائية.
- إعادة المعالجة idempotent ولا تنشئ EDI documents مكررة.
- توجد اختبارات fixtures رسمية لكل نوع فاتورة وتعديل وإلغاء.

## 7. Inventory: Barcode وPackaging وRoutes وScrap وTraceability

### 7.1 Barcode وGS1

#### الموجود

يوجد barcode في المنتج، كما أن المشروع يملك lots وserial/tracking modes ومسارات مخزون.

#### الناقص

وجود barcode كحقل لا يعني وجود تطبيق Barcode. ينقص:

- barcode rules وnomenclature.
- GS1 parsing للـ GTIN والlot والserial والكمية والتاريخ.
- scanner endpoint يطبق العملية بحسب نوع الشاشة.
- barcode-driven reservation وvalidate picking.
- رسائل أخطاء تمنع مسح منتج في موقع خاطئ.

#### شرط الإكمال

نفس scan sequence يجب أن يعطي نفس النتيجة المتوقعة في Odoo: تحديد المنتج، lot/serial، الكمية، الموقع، والحركة، مع منع serial المكرر.

### 7.2 Packaging

#### الموجود

`StockMoveLine` يحتوي `PackageID`.

#### التناقض

هذا الحقل وحده لا يمثل package. لا توجد دورة إنشاء حزمة أو نقلها أو فتحها أو حساب وزنها أو تاريخها.

#### المطلوب

- `stock.package` و`stock.package.type`.
- parent/child packages.
- package contents وlocation وowner.
- put-in-pack وremove-from-pack.
- وزن وحجم محسوبان من المنتجات.
- package history وtraceability.

### 7.3 Routes وRules وProcurement

#### الموجود

المشروع يملك `ProcurementGroup` وorderpoints وprocurement، كما يوجد `RouteID` في سطر البيع.

#### التناقض

`RouteID` منفرد لا يقرر وحده هل الطلب يشتري أو يصنع أو ينقل. سلوك Odoo يحتاج `stock.rule` مع شروط المنتج والموقع والشركة والمهلة ومجموعة التوريد.

#### المطلوب

- route model وقواعد pull/push.
- اختيار rule deterministic مع سبب القرار.
- propagation للـ origin وgroup من SO/PO/MO إلى picking/moves.
- lead times وdate propagation.
- منع إنشاء طلب مكرر عند إعادة تشغيل scheduler.

### 7.4 Scrap

#### الموجود

يوجد scrap location، وتوجد حركات تصحيح مخزون.

#### النقص

لا توجد وثيقة `stock.scrap` مستقلة لها source location وscrap location وreason وlot/serial وquantity وstate وvaluation link.

#### شرط الإكمال

تأكيد scrap ينشئ حركة من المصدر إلى موقع الخردة، يحجز الكمية الصحيحة، يمنع كمية سالبة أو serial مكرر، ويربط أثر التقييم والحركة بالوثيقة.

### 7.5 Traceability

#### الموجود

lot/serial وmove lines موجودة.

#### الناقص

- upstream/downstream graph.
- ربط purchase receipt بالتصنيع والبيع.
- genealogy للمواد والمنتج النهائي.
- package history وrecall path.
- تقارير من العميل إلى المورد والعكس.

#### شرط الإكمال

إدخال lot أو serial في تقرير traceability يعرض كل الحركات والوثائق السابقة واللاحقة بترتيب زمني، مع احترام الشركة والصلاحيات.

## 8. Sales وPurchase وStock وAccounting

### 8.1 Sales

#### الموجود

Sale order states موجودة، مع خطوط وأسعار وخصومات وضرائب وفاتورة وربط delivery وprocurement group، وتكامل تحويل CRM opportunity إلى sale order.

#### النقص

- price rules حسب العميل والكمية والفترة والعملة.
- sales teams وassignment وUTM وscoring.
- advance payment وpayment link.
- delivered/invoiced quantities الناتجة عن moves/invoices، لا قيم إدخال مستقلة.
- portal وchatter وactivities.
- refunds وreturns وربطها بالفاتورة والمخزون.

#### التناقض المحتمل

الدالة `ComputeAmounts` تطبق tax rates جمعياً على نفس الأساس. Odoo قد يطبق taxes بتسلسل، price-included، repartition، group taxes، rounding per line أو globally. يجب تحديد tax engine واحد مطابق قبل إعلان تطابق الإجماليات.

### 8.2 Purchase

#### الموجود

RFQ/PO states، vendor bills، receipt links، requisitions، alternative groups، وتكامل أساسي مع stock/accounting موجودة.

#### النقص

- supplier info وvendor pricelist بفعالية كاملة في اختيار السعر.
- purchase agreements/tenders.
- approval levels والحدود المالية.
- three-way matching بين PO وreceipt وvendor bill.
- backorders وpartial receipts.
- اختلافات الكمية والسعر والفاتورة والإشعارات.

#### شرط الإكمال

لا تتحول `InvoiceStatus` إلى `invoiced` إلا بناءً على الكمية المفوترة المعتمدة، ولا تتحول `ReceiptStatus` إلا من كميات الحركات المستلمة، مع دعم partial وbackorder.

### 8.3 التكاملات العابرة

يجب اعتماد مصدر واحد للحقيقة:

| القيمة | المصدر الصحيح |
|---|---|
| delivered quantity | stock moves المنفذة المرتبطة بسطر البيع |
| received quantity | incoming moves المنفذة المرتبطة بسطر الشراء |
| invoiced quantity | account move lines المرتبطة بالسطر |
| inventory quantity | quants الناتجة عن الحركات، لا تعديل مباشر في الطلب |
| procurement origin | procurement group وorigin chain |
| stock valuation | valuation layers الناتجة عن الحركات |

أي API يسمح بتعديل هذه القيم مباشرة دون إعادة حساب من المصدر سيخلق تناقضاً مع Odoo.

## 9. MRP

### الموجود

المشروع يملك BOM وBOM lines وoperations وproduction وworkcenter وworkorder وunbuild، مع ربط بالمخزون والمنتجات وبعض المحاسبة.

### الناقص

- MRP scheduler وحساب الاحتياج.
- routes وmanufacture rules.
- by-products.
- scrap أثناء الإنتاج.
- backorders وsplit production.
- serial/lot generation وربط genealogy.
- subcontracting.
- quality checks.
- WIP وmaterial/labor overhead accounting.
- استهلاك فعلي يختلف عن الاستهلاك النظري مع variance.

### التناقض التوثيقي

وجود كود MRP فعلي مع بقاء خطة المرحلة مصاغة كأن التنفيذ لم يبدأ يسبب التباساً في حالة المشروع. يجب تحديث الوثيقة الأصلية إلى: implemented baseline، partial parity، remaining gaps.

### شرط الإكمال

تأكيد MO ينشئ الحركات المطلوبة مرة واحدة، والحجز يتبع availability، والإنتاج الجزئي ينشئ backorder، وكل consumption/by-product يرتبط بالـ MO والـ lot والـ valuation.

## 10. Project وCRM وHR

### 10.1 Project

#### الموجود

Projects، tasks، stages، tags، milestones، dependencies.

#### الناقص

- collaborators وroles وصلاحيات المشروع.
- recurring tasks وtask templates.
- project updates ومؤشرات الحالة.
- timesheets والتكلفة والفوترة التحليلية.
- calendar integration وportal sharing.
- chatter وactivities على المشروع والمهمة.

#### شرط الإكمال

إسناد المهمة، الوقت المسجل، التكلفة، والفاتورة يجب أن تشترك في نفس company/project/task security context، لا أن تكون جداول منفصلة بلا أثر محاسبي.

### 10.2 CRM

#### الموجود

Lead/opportunity، stages، tags، lost reasons، partner conversion، وتكامل محدود مع Sale.

#### الناقص

- `crm.team` وأعضاء الفريق وقواعد التوزيع.
- assignment rules وterritories.
- lead scoring وpredictive frequency.
- recurring plans وactivity schedules.
- merge opportunities وmass conversion.
- UTM campaigns وsource/medium.
- calendar/mail/chatter وemail aliases.

#### شرط الإكمال

تحويل opportunity إلى sale يجب أن يكون انتقالاً ذرياً يحفظ origin، team، salesperson، campaign، activities، partner، ويمنع إنشاء order مكرر عند إعادة الطلب.

### 10.3 HR

#### الموجود

Employee، department، job، attendance، leave، leave allocation، وبعض overtime.

#### الناقص

- contracts وresource calendars.
- work schedules وwork entries.
- recruitment وskills وappraisal.
- timesheets وربطها بالمشاريع.
- payroll-facing semantics.
- employee chatter وactivity plans.

#### التناقضات المطلوب منعها

- تسجيل حضور خارج جدول العمل دون سياسة واضحة.
- إنشاء إجازة تتداخل مع حضور أو إجازة أخرى دون قرار مطابق لـ Odoo.
- حساب الأيام دون timezone/company calendar.
- إنشاء overtime دون مصدر حضور أو قاعدة موافقة.

## 11. Fleet وMaintenance وLoyalty

### 11.1 Fleet

#### الموجود

Vehicles، models، brands، tags، states، odometers، assignment logs، services، contracts.

#### الناقص

- ربط الموظف والمستخدم والشركة والجهة المالكة بدقة.
- تكلفة المركبة التحليلية من fuel/service/contract.
- تنبيهات انتهاء العقد والخدمة عبر Mail/Activity.
- تقارير cost per vehicle وodometer history.
- تكامل Fleet مع Maintenance وExpenses وAccounting.

### 11.2 Maintenance

#### الموجود

Equipment، categories، teams، stages، requests، وجدولة follow-up activity.

#### الناقص

- preventive scheduling مبني على date/odometer/usage.
- calendar/resource capacity.
- قطع الغيار واستهلاك المخزون.
- تكلفة الصيانة وقيدها المحاسبي.
- ربط workcenter/MRP.
- MTBF وMTTR وavailability.
- chatter/followers.

#### ملاحظة مهمة

لا ينبغي إنشاء بنية مختلفة عن Odoo لمجرد التشابه الاسمي. إذا كان Odoo يضع التكرار على request أو equipment، يجب تثبيت ذلك بعد قراءة model contract قبل إنشاء `maintenance.plan` مستقل.

### 11.3 Loyalty

#### الموجود

Programs، rules، rewards، cards/coupons، points history، earn/redeem، وربط أساسي مع Sales.

#### الناقص

- POS وWebsite/eCommerce.
- coupon issuance/distribution.
- refund وpartial return وإرجاع النقاط.
- payment programs.
- customer portal wallet.
- Mail/SMS campaigns.
- منع الاستبدال المزدوج عند concurrent requests.

#### شرط الإكمال

كل تغيير نقاط يجب أن يكون ledger entry غير قابل للتلاعب، مرتبطاً بالطلب والعملية والسبب، مع reversal عند الإلغاء أو refund بحسب حالة الطلب.

## 12. التناقضات العامة التي يجب حسمها

### 12.1 Domain مقابل Use Case مقابل Storage

يجب ألا يسمح domain بقيمة ثم يرفضها use case، أو يقبلها memory repository ويرفضها PostgreSQL. يلزم contract tests مشتركة لكل repository implementation.

### 12.2 الحالات والانتقالات

كل حالة يجب أن تملك جدول انتقالات واضحاً:

- من draft إلى sent/confirmed.
- من confirmed إلى done/cancelled.
- ما الذي يمكن عكسه؟
- هل الإلغاء ينشئ reversal أم يغير flag فقط؟
- هل إعادة الطلب idempotent؟

### 12.3 المال والتقريب والضرائب

يجب تثبيت:

- دقة العملة.
- rounding per line أو per tax أو global.
- price-included taxes.
- tax groups وrepartition.
- currency conversion date/rate.

لا يكفي استعمال `float64` وإجراء تقريب محلي إذا كانت النتائج المحاسبية يجب أن تطابق Odoo.

### 12.4 الشركة والصلاحيات

كل query وmutation يجب أن يطبق company scope وrecord rules في طبقة واحدة قابلة للاختبار. لا يكفي وجود `CompanyID` في struct إذا كان الاستعلام أو الذاكرة يسمحان بإرجاع سجل شركة أخرى.

### 12.5 Memory وPostgreSQL

الـ Memory repositories المستخدمة في الاختبارات يجب أن تطبق نفس القيود والـ cascades والـ uniqueness الموجودة في PostgreSQL. خلاف ذلك ستنجح اختبارات لا تمثل الإنتاج.

## 13. خطة الإكمال الصارم

### المرحلة A: تثبيت العقود

1. توثيق state machines لكل Sale/Purchase/Stock/Payment/MRP/EDI.
2. توحيد validation وrounding وcompany scope.
3. إنشاء contract tests مشتركة بين Memory وPostgreSQL.
4. تحديث وثائق المراحل لتفرق بين implemented وpartial وplanned.

### المرحلة B: الطبقة المشتركة

1. Mail thread/followers/subtypes/tracking/templates.
2. Attachments وportal access.
3. Automation scheduler وactivity execution log.
4. Outbox/idempotency/retry للمراسلات والتكاملات.

### المرحلة C: سلسلة الإمداد

1. Package وPackage Type.
2. Barcode/GS1 service.
3. Routes/Rules/Scheduler.
4. Scrap وtraceability graph.
5. source-of-truth quantities وbackorders/reservations.

### المرحلة D: المال والتكامل الخارجي

1. Payment providers وtransactions وwebhooks.
2. Tax engine مطابق للعقد المطلوب.
3. EDI validated documents والتوقيع وQR والإرسال.
4. refund/capture/void والتسويات.

### المرحلة E: إكمال التطبيقات

1. Sales/Purchase/MRP integration.
2. Project/CRM/HR ecosystem.
3. Fleet/Maintenance/Loyalty integrations.
4. Mass mailing/SMS/Marketing/Live Chat حسب أولوية المنتج.

## 14. بوابة الجودة قبل إعلان التكافؤ

لا تعتبر أي ميزة مكتملة حتى تحقق جميع البنود التالية:

- Domain invariants موثقة ومختبرة.
- انتقالات الحالة تمنع المسارات غير الصحيحة.
- العمليات المركبة ذرية وقابلة لإعادة المحاولة بأمان.
- Memory وPostgreSQL متطابقان في السلوك.
- company scope وrecord rules مطبقة في القراءة والكتابة.
- الأحداث والرسائل والآثار المحاسبية مرتبطة بالسجل الأصلي.
- التكامل الخارجي يتحقق من التوقيع ويعالج التكرار والفشل.
- توجد اختبارات end-to-end للسيناريو التجاري، لا اختبارات CRUD فقط.
- لا توجد TODO أو mock أو placeholder في مسار إنتاجي معلن كمكتمل.
- يمكن مقارنة النتيجة مع Odoo لنفس السيناريو والبيانات والعملة والتواريخ.

## 15. الخلاصة

المشروع يملك أساساً قوياً في عدة مجالات، لكن الفجوة الأساسية مع Odoo ليست عدد الجداول. الفجوة هي أن Odoo يربط كل مجال بدورة عمل مشتركة: رسائل ومتابعون، صلاحيات، حالات، أنشطة، مخزون، محاسبة، جدولة، وتكاملات خارجية.

لذلك يجب تنفيذ الإكمال على شكل عقود سلوكية وتكاملات قابلة للاختبار. إضافة endpoint أو حقل جديد دون إغلاق مصدر الحقيقة، state machine، idempotency، والصلاحيات ستزيد مساحة التناقض بدلاً من تحقيق parity.
