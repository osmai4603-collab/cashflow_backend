# معايير الترقيم والفلترة والفرز (Pagination, Filtering & Sorting)

في قواعد البيانات التي تحتوي على ملايين السجلات، يؤدي طلب قائمة السجلات دون ترقيم مقيد إلى استنزاف الذاكرة وانهيار المعالج.

---

## 1. الترقيم الآمن بالصفحات (Offset/Limit Pagination)

- **`limit`**: عدد السجلات في الصفحة.
  - القيمة الافتراضية: `20`.
  - الحد الأقصى الصارم: `100`. (إذا أرسل العميل `limit=10000`، يتم قصها تلقائياً إلى `100`).
- **`offset`**: إزاحة البداية ($\text{offset} = (\text{page} - 1) \times \text{limit}$).

### البيانات الوصفية في `meta`

```json
{
  "meta": {
    "page": 2,
    "limit": 20,
    "total_records": 1540,
    "total_pages": 77
  }
}
```

---

## 2. الترقيم بالعلامات (Cursor-Based Pagination)

للجداول الضخمة جداً أو الخلاصات الحية (Feeds):

- استخدام حقل فريد ورتيب (مثل `created_at` أو `id`).
- الاستعلام: `WHERE created_at < $cursor ORDER BY created_at DESC LIMIT $limit`.
- تجنب تكلفة `OFFSET` العالية في PostgreSQL عندما تكبر الإزاحة.

---

## 3. حماية الفرز من ثغرات SQL Injection (Whitelisted Sorting)

> [!CAUTION]
> **لا تمرر أبداً قيمة `sort` من الـ Query String مباشرة إلى جملة `ORDER BY` في SQL.**

### النمط الآمن

استخدام قائمة بيضاء (Whitelist) معتمدة:

```go
var allowedSortFields = map[string]string{
    "date":   "created_at",
    "amount": "amount",
    "status": "status",
}
```

إذا طلب العميل حقلاً غير موجود في القائمة، يتم التراجع إلى الترتيب الافتراضي الآمن.
