# نماذج NIST للتحكم بالوصول القائم على الأدوار والرسم البياني الهرمي DAG في Go

توضح هذه الوثيقة النمذجة الرياضية الرسمية للمستويات الأربعة القياسية لنموذج RBAC وفق معايير **NIST / ANSI INCITS 359-2012**، وكيفية تطبيقها باستخدام هياكل Go الاصطلاحية وعالية الكفاءة.

---

## 1. النماذج الرسمية المعتمدة من NIST

```text
       ┌────────────────────────┐
       │   4. النموذج المتناظر   │
       │ (الاستعلام ثنائي الاتجاه)│
       └───────────▲────────────┘
                   │
       ┌───────────┴────────────┐
       │   3. النموذج المقيد    │
       │(الفصل بين الواجبات SSD)│
       └───────────▲────────────┘
                   │
       ┌───────────┴────────────┐
       │    2. النموذج الهرمي   │
       │  (وراثة الأدوار عبر DAG)│
       └───────────▲────────────┘
                   │
       ┌───────────┴────────────┐
       │    1. النموذج الأساسي   │
       │(المستخدمون، الأدوار،..)│
       └────────────────────────┘
```

### المستوى 1: نموذج RBAC الأساسي (Core / Flat RBAC)
- **المستخدمون ($U$)**: الكيانات (المستخدمون البشر، حسابات النظام، الخدمات المصغرة) التي تطلب الوصول.
- **الأدوار ($R$)**: المسميات الوظيفية التي تجمع الواجبات والمسؤوليات.
- **الصلاحيات ($P = OP \times OBJ$)**: اقتران العملية (قراءة، تعديل، اعتماد) بكائن محدد (فواتير، مستخدمين، طلبات).
- **إسناد المستخدمين للأدوار ($UA \subseteq U \times R$)**: علاقة متعدد-إلى-متعدد تربط المستخدمين بالأدوار.
- **إسناد الصلاحيات للأدوار ($PA \subseteq P \times R$)**: علاقة متعدد-إلى-متعدد تربط الصلاحيات بالأدوار.

### المستوى 2: نموذج RBAC الهرمي (Hierarchical RBAC)
يقدم علاقة ترتيب جزئي ($\succeq$) على مجموعة الأدوار $R$.
إذا كان $r_{senior} \succeq r_{junior}$، فإن:
$$P(r_{senior}) \supseteq P(r_{junior})$$
المستخدم المسند إلى $r_{senior}$ يكتسب تلقائياً كافة صلاحيات $r_{senior}$ المباشرة بالإضافة لكافة الصلاحيات الموروثة من $r_{junior}$.

### المستوى 3: نموذج RBAC المقيد (Constrained RBAC - الفصل بين الواجبات)
يفرض قيوداً أمنية تمنع تركز الصلاحيات الحساسة:
- **الفصل الاستاتيكي بين الواجبات (SSD)**: قيد يمنع إسناد دورين متعارضين لنفس المستخدم وقت الإنشاء:
  $$\forall u \in U, \quad \{r_1, r_2\} \subseteq UA(u) \implies \text{SSD\_Conflict}(r_1, r_2) = \text{false}$$
- **الفصل الديناميكي بين الواجبات (DSD)**: قيد يمنع تفعيل دورين متعارضين معاً في نفس الجلسة أو المعاملة التشغيلية:
  $$\forall s \in S(u), \quad \{r_1, r_2\} \subseteq ActiveRoles(s) \implies \text{DSD\_Conflict}(r_1, r_2) = \text{false}$$

### المستوى 4: نموذج RBAC المتناظر (Symmetric RBAC)
يفرض تناظر الاستعلام في النظام:
- **الاستعلام الأمامي**: إعطاء مستخدم أو دور واسترجاع كافة الصلاحيات المصرح له بها.
- **الاستعلام العكسي**: إعطاء صلاحية أو مورد حساس واسترجاع كافة الأدوار والمستخدمين الذين يمتلكون حق الوصول إليها، وهو شرط أساسي للتدقيق الأمني والامتثال الرقابي.

---

## 2. التطبيق الاصطلاحي للرسم البياني الهرمي (DAG) في Go

في Go، يتم نمذجة الرسم البياني الموجه غير الدائري (DAG) لوراثة الأدوار بكفاءة عبر الخرائط القياسية (Hash Maps):

```go
package rbac

import "fmt"

type RoleDAG struct {
	// parentToChildren يربط الأدوار الأعلى بالأدوار الأدنى الموروثة
	parentToChildren map[Role][]Role
}

func NewRoleDAG() *RoleDAG {
	return &RoleDAG{
		parentToChildren: make(map[Role][]Role),
	}
}

// AddInheritance يسجل أن الدور الأعلى senior يرث من junior مع فحص الحلقات التكرارية
func (dag *RoleDAG) AddInheritance(senior, junior Role) error {
	if dag.hasPath(junior, senior) {
		return fmt.Errorf("اكتشاف حلقة وراثة تكرارية: %s -> %s سيخلق دورة مغلقة", senior, junior)
	}
	dag.parentToChildren[senior] = append(dag.parentToChildren[senior], junior)
	return nil
}

// hasPath يستخدم خوارزمية البحث بالعمق (DFS) للتحقق من عدم وجود مسارات عكسية
func (dag *RoleDAG) hasPath(start, target Role) bool {
	visited := make(map[Role]bool)
	var dfs func(current Role) bool
	dfs = func(current Role) bool {
		if current == target {
			return true
		}
		visited[current] = true
		for _, next := range dag.parentToChildren[current] {
			if !visited[next] {
				if dfs(next) {
					return true
				}
			}
		}
		return false
	}
	return dfs(start)
}

// ResolveAllInherited يستخرج كافة الأدوار الموروثة المباشرة وغير المباشرة
func (dag *RoleDAG) ResolveAllInherited(roles []Role) []Role {
	allRoles := make(map[Role]bool)
	var walk func(r Role)
	walk = func(r Role) {
		if allRoles[r] {
			return
		}
		allRoles[r] = true
		for _, child := range dag.parentToChildren[r] {
			walk(child)
		}
	}
	for _, r := range roles {
		walk(r)
	}

	result := make([]Role, 0, len(allRoles))
	for r := range allRoles {
		result = append(result, r)
	}
	return result
}
```

---

## 3. الاستعلام العكسي في النموذج المتناظر (Symmetric RBAC Auditing)

للامتثال لمتطلبات المستوى الرابع، يوفر المحرك إمكانية الاستعلام العكسي للتدقيق الأمني:

```go
// WhoCanPerform يسترجع كافة الأدوار التي تملك حق تنفيذ صلاحية معينة
func (e *Engine) WhoCanPerform(targetPerm Permission) []Role {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var capableRoles []Role
	for role := range e.rolePerms {
		if e.HasPermission([]Role{role}, targetPerm) {
			capableRoles = append(capableRoles, role)
		}
	}
	return capableRoles
}
```
