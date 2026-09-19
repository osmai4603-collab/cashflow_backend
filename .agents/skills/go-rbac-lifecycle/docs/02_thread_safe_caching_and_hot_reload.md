# التخزين المؤقت المتزامن عالي الأداء والتحديث اللحظي في Go

يتم فحص الصلاحيات وتقييمها مع كل طلب شبكي قادم تقريباً. إن استعلام قاعدة البيانات العلائقية في كل فحص هو نمط مضاد يخلق عنق زجاجة حاد. توضح هذه الوثيقة كيفية تطبيق تخزين مؤقت في الذاكرة بزمن استجابة أقل من الميكروثانية مع دعم التحديث اللحظي للسياسات دون أي توقف في الخدمة (Zero-Downtime Hot-Reloading).

---

## 1. أنماط التزامن: المقارنة بين `sync.RWMutex` و `atomic.Pointer`

اعتماداً على معدل تعديل تعريفات الأدوار، توفر Go نمطين اصطلاحيين لإدارة التزامن:

```text
النمط أ: قفل القراءة والكتابة sync.RWMutex (تزامن مرتفع مع تحديثات دورية)
┌────────────────────────────────────────────────────────┐
│ مسارات قراءة متعددة (RLock) ─────────► ذاكرة مشتركة     │
│ مسار كتابة مفرد (Lock) ──────────────► ذاكرة مشتركة     │
└────────────────────────────────────────────────────────┘

النمط ب: المؤشرات الذرية atomic.Pointer (قراءة كثيفة خالية من تنازع الأقفال)
┌────────────────────────────────────────────────────────┐
│ مسارات القراءة ───────────────────────► لقطة الذاكرة أ   │
│ مسار الكتابة يجهز لقطة جديدة ب في الذاكرة             │
│ مسار الكتابة يستبدل المؤشر ذرياً: أ ────► ب            │
└────────────────────────────────────────────────────────┘
```

### النمط أ: استخدام `sync.RWMutex` (المحرك القياسي المدمج)

```go
type Engine struct {
	mu          sync.RWMutex
	rolePerms   map[Role]map[Permission]bool
	inheritance map[Role][]Role
}

func (e *Engine) HasPermission(roles []Role, required Permission) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	// قراءة محمية ومتزامنة
	return e.evaluate(roles, required)
}
```

### النمط ب: الاستبدال الذري الخالي من الأقفال عبر `atomic.Pointer`

للأنظمة ذات الإنتاجية الفائقة (أكثر من $100,000\text{ طلب/ثانية}$) حيث يكون التنازع على الأقفال أثناء تحديث السياسات غير مقبول، يتم استخدام `sync/atomic.Pointer`:

```go
package rbac

import (
	"sync/atomic"
)

type PolicySnapshot struct {
	RolePerms   map[Role]map[Permission]bool
	Inheritance map[Role][]Role
}

type AtomicEngine struct {
	current atomic.Pointer[PolicySnapshot]
}

func NewAtomicEngine(initial *PolicySnapshot) *AtomicEngine {
	e := &AtomicEngine{}
	e.current.Store(initial)
	return e
}

// HasPermission يقيم الصلاحيات بصفر أقفال
func (e *AtomicEngine) HasPermission(roles []Role, required Permission) bool {
	snap := e.current.Load()
	if snap == nil {
		return false // المنع الافتراضي Default Deny
	}
	return evaluateSnapshot(snap, roles, required)
}

// Reload يستبدل كامل لقطة السياسات ذرياً دون حظر أي من مسارات القراءة
func (e *AtomicEngine) Reload(newSnapshot *PolicySnapshot) {
	e.current.Store(newSnapshot)
}
```

---

## 2. استراتيجيات إطلاق التحديث اللحظي (Hot-Reloading Triggers)

في البيئات الإنتاجية، تتغير السياسات عندما يقوم المسؤولون بإضافة صلاحيات أو تعديل أدوار. يجب أن يُعاد تحميل الذاكرة الحية دون الحاجة لإعادة تشغيل الخدمة:

1. **الاستقصاء الدوري ومراقبة التغيير (CDC / Polling)**:
   يقوم عامل خلفي (Worker) بالاستعلام عن جداول `roles` و `permissions` كل $N$ ثانية، ومقارنة الطابع الزمني لآخر تحديث، ثم إعادة بناء لقطة الذاكرة.
2. **ناقل الرسائل والأحداث الموزعة (Pub/Sub Notification)**:
   عندما يقوم المسؤول بتعديل سياسة عبر لوحة التحكم، يتم بث رسالة عبر قناة إشعارات داخلية أو موضوع في Redis/NATS (`rbac.policy.updated`)، مما يدفع كافة مثيلات الخوادم لتحديث لقطة الذاكرة المؤقتة فورياً.
3. **إشارات نظام التشغيل (POSIX SIGHUP Reload)**:
   الاستماع لإشارة `syscall.SIGHUP` لإعادة قراءة ملفات التكوين محلياً:

```go
func ListenForPolicyReloadSignal(engine *AtomicEngine, loader func() (*PolicySnapshot, error)) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for range c {
			newSnap, err := loader()
			if err == nil {
				engine.Reload(newSnap)
			}
		}
	}()
}
```
