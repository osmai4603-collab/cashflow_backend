# البحث المرجعي الشامل: مفهوم وخصائص ACID في أنظمة إدارة قواعد البيانات

---

## 1. المقدمة والخلفية التاريخية (Historical Context & Foundations)

في علوم الحاسوب وهندسة البرمجيات، تُعد **المعاملة (Transaction)** وحدة منطقية لمعالجة البيانات تتكون من عملية واحدة أو سلسلة من العمليات المتتابعة (قراءة، كتابة، تعديل، حذف) التي يجب أن تُعامل ككتلة ذرية واحدة لحماية البيانات من التلف أو التناقض.

### محطات تاريخية تأسيسية

1. **جيم غراي (Jim Gray - 1981):**
   وضع عالم الحاسوب الحائز على جائزة تورينغ **جيم غراي** حجر الأساس النظري لمعاملات قواعد البيانات في ورقته البحثية البارزة:
   > *"The Transaction Concept: Virtues and Limitations"* (IEEE / Tandem Technical Report, 1981).
   قدم غراي تعريف المعاملة كتحول موثوق بين الحالات المتسقة، وحدد مبادئ الالتزام (Commit) والتراجع (Abort/Rollback).

2. **ثيو هاردر وأندرياس رويتر (Theo Härder & Andreas Reuter - 1983):**
   في ورقتهما الأكاديمية التاريخية المنشورة في دورية *ACM Computing Surveys*:
   > *"Principles of Transaction-Oriented Database Recovery"* (ACM Computing Surveys, Vol. 15, No. 4, Dec 1983).
   قام الباحثان رسميًا بصياغة الاختصار الشهير **ACID** لتوحيد وتصنيف الخصائص الأربع الجوهرية التي تضمن سلامة واعتمادية قواعد البيانات أثناء المعالجة المتزامنة وعند حدوث الأعطال:
   * **A - Atomicity** (الذرية / عدم التجزئة)
   * **C - Consistency** (الاتساق / الصحة التوافقية)
   * **I - Isolation** (العزل / استقلالية المعاملات المتزامنة)
   * **D - Durability** (الديمومة / استمرارية البيانات)

3. **خوارزمية التعافي ARIES (C. Mohan et al. - IBM 1992):**
   > *"ARIES: A Transaction Recovery Method Supporting Fine-Granularity Locking and Partial Rollbacks Using Write-Ahead Logging"* (ACM TODS, 1992).
   طوّرت شركة IBM خوارزمية ARIES التي أصبحت النموذج الصناعي القياسي المعتمد في معظم محركات قواعد البيانات الحديثة (مثل DB2 و Microsoft SQL Server و PostgreSQL و SQLite) لتطبيق مبادئ الذرية والديمومة بكفاءة عبر سجلات WAL.

---

## 2. التحليل الهيكلي لخصائص ACID وآليات عملها هندسياً

### أ. الذرية (Atomicity) — قاعدة "الكل أو لا شيء" (All or Nothing)

#### المفهوم

تفرض الذرية أن جميع العمليات داخل المعاملة تُنفذ بنجاح تام وتُثبّت في قاعدة البيانات (`Commit`)، أو في حال حدوث أي فشل في أي جزء من أجزاء المعاملة (خطأ منطقي، انهيار برمجيات، انقطاع التيار، أو انقطاع الاتصال بالشبكة)، يتم إلغاء المعاملة بالكامل والتراجع عن كافة التغييرات الجزئية التي قامت بها (`Rollback`)، لتعود قاعدة البيانات إلى الحالة الدقيقة التي كانت عليها قبل بدء المعاملة.

#### سيناريو توضيحي

في نظام التحويل المالي المصرفي:

```sql
BEGIN TRANSACTION;
  UPDATE accounts SET balance = balance - 500 WHERE account_id = 'A';
  UPDATE accounts SET balance = balance + 500 WHERE account_id = 'B';
COMMIT;
```

إذا انقطع الخادم أو فشل الاتصال بعد العملية الأولى وقبل الثانية، تضمن الذرية التراجع عن خصم الـ 500 من الحساب (A)، مانعةً اختفاء الأموال دون وصولها للحساب (B).

#### آليات التنفيذ الهندسية في محركات قواعد البيانات

1. **سجل المعاملات المسبق (Write-Ahead Logging - WAL):**
   قاعدة جوهرية في محركات التخزين تقتضي كتابة وحفظ سجلات التعديل (Log Records) في وحدة التخزين الدائمة قبل أن يُسمح بكتابة صفحات البيانات الفعلية في الذاكرة (Buffer Pool) إلى القرص الصلب.
2. **سجلات التراجع (Undo Logs / Rollback Segments):**
   تخزين نسخ من الحالة السابقة للبيانات قبل التعديل. عند استدعاء أمر `ROLLBACK` أو إعادة التشغيل بعد الانهيار، يقوم محرك التخزين (مثل MySQL InnoDB أو Oracle) بقراءة سجلات Undo لعكس التغييرات واستعادة القيم الأصلية.
3. **تقنية صفحات الظل (Shadow Paging):**
   تعديل البيانات في صفحات بديلة في الذاكرة، ولا يتم ربط المؤشرات بالصفحات الجديدة إلا بعد تأكيد اكتمال المعاملة بالكامل.

---

### ب. الاتساق (Consistency) — صيانة القواعد والمتغيرات الحاكمة

#### المفهوم

يضمن الاتساق أن المعاملة تنقل قاعدة البيانات من **حالة صالحة (Valid State)** إلى **حالة صالحة أخرى**، دون خرق أي من القواعد الصارمة والقيود المعرفة مسبقاً في مخطط قاعدة البيانات (Schema) أو القواعد المنطقية لتطبيق الأعمال (Business Invariants).

#### القيود المحمية بموجب الاتساق

* **القيود الهيكلية للمخطط (Database Schema Constraints):**
  * قيود المفتاح الأساسي (`PRIMARY KEY`): منع التكرار وضمان هوية السجل.
  * قيود التكامل المرجعي (`FOREIGN KEY`): منع السجلات اليتيمة.
  * قيود التحقق والقيم الفريدة (`CHECK` و `UNIQUE`).
  * قيود عدم القبول بالفراغ (`NOT NULL`).
* **ثوابت ومتغيرات العمليات التجارية (Business Invariants):**
  * مثلاً: "يجب ألا يقل مجموع الأصول عن مجموع الخصوم"، أو "يجب ألا ينخفض رصيد العميل عن الصفر في حسابات السحب العادي".

#### تمييز علمي جوهري: اتساق ACID مقابل اتساق CAP Theorem

* **في ACID:** الاتساق يعني **صحة البيانات وسلامة القيود (Correctness & Integrity)**. المسؤولية هنا مشتركة بين محرك قاعدة البيانات (فرض القيود الهيكلية) والمطور (كتابة معاملات صحيحة منطقياً).
* **في نظرية CAP (CAP Theorem):** الاتساق يعني **الخطية (Linearizability / Single-Copy Consistency)**؛ أي أن كل قراءة من أي عقدة في النظام الموزع يجب أن ترجع أحدث قيمة تمت كتابتها، وكأن النظام يعمل على جهاز واحد.

---

### ج. العزل (Isolation) — إدارة التزامن وتفادي التداخل (Concurrency Control)

#### المفهوم

يضمن العزل أن المعاملات التي تعمل بالتوازي لا تتداخل ولا تؤثر نتائجها غير المكتملة على بعضها البعض. يجب أن تكون نتيجة تنفيذ عدة معاملات متزامنة مطابقة تماماً للنتيجة التي كانت ستتحقق لو تم تنفيذها بالتسلسل واحدة تلو الأخرى (`Serial Execution`).

#### الظواهر الشاذة الناتجة عن غياب العزل الكامل (Concurrency Anomalies)

1. **القراءة القذرة (Dirty Read - G1):**
   معاملة تقرأ بيانات تم تعديلها بواسطة معاملة متزامنة أخرى لم تنتهِ بعد ولم تقم بالتثبيت (`Uncommitted`). إذا تراجعت المعاملة الثانية (`Aborted`)، تكون المعاملة الأولى قد بنت قراراتها على بيانات غير موجودة أصلاً.
2. **القراءة غير المتكررة (Non-Repeatable / Fuzzy Read - G2a):**
   معاملة تقرأ نفس السجل مرتين خلال جلسة عملها، فتحصل على قيمتين مختلفتين لأن معاملة متزامنة قامت بتعديل هذا السجل وتثبيته في الفترة بين القراءتين.
3. **القراءة الوهمية (Phantom Read - A3):**
   معاملة تنفذ استعلاماً شرطياً بنطاق معين (مثل: `SELECT * FROM users WHERE status = 'active'`) فتسترجع 10 صفوف. أثناء ذلك، تقوم معاملة أخرى بإدراج مستخدم جديد نشط وتثبيته. عند إعادة المعاملة الأولى لنفس الاستعلام، تظهر 11 صفاً.
4. **انحراف الكتابة (Write Skew):**
   حالة متقدمة تحدث عندما تقرأ معاملتان نفس البيانات ثم تقومان بتعديل سجلات منفصلة بطريقة تخرق قيداً تكاملياً مشتركاً (كأن يُسمح لطبيبين بأخذ إجازة معاً بشرط بقاء طبيب واحد مناوب على الأقل).

#### مستويات العزل المعتمدة في المعيار الدولي (ANSI/ISO SQL-92 Standard)

| مستوى العزل (Isolation Level) | القراءة القذرة (Dirty Read) | القراءة غير المتكررة (Non-Repeatable Read) | القراءة الوهمية (Phantom Read) |
| :--- | :---: | :---: | :---: |
| **Read Uncommitted** | ممكنة (Allowed) | ممكنة (Allowed) | ممكنة (Allowed) |
| **Read Committed** | **ممنوعة** (Prevented) | ممكنة (Allowed) | ممكنة (Allowed) |
| **Repeatable Read** | **ممنوعة** (Prevented) | **ممنوعة** (Prevented) | ممكنة (Allowed) |
| **Serializable** | **ممنوعة** (Prevented) | **ممنوعة** (Prevented) | **ممنوعة** (Prevented) |

> **ملاحظة أكاديمية:** في عام 1995، نشر كبار علماء قواعد البيانات ورقة نقدية شهيرة:  
> *"A Critique of ANSI SQL Isolation Levels"* (Berenson et al. - ACM SIGMOD 1995)، أثبتوا فيها أن تصنيف ANSI SQL-92 يعاني من قصور، وقدموا تعريفاً دقيقاً لظواهر أخرى مثل *Snapshot Isolation* و *Write Skew*.

#### التقنيات الهندسية لتنفيذ العزل في المحركات الحديثة

1. **القفل التشاؤمي ثنائي المراحل (Two-Phase Locking - 2PL / Strict 2PL):**
   حجز أقفال مشتركة (Shared Locks) للقراءة وأقفال حصرية (Exclusive Locks) للكتابة، مع مرحلة تنامي (Growing Phase) لجمع الأقفال ومرحلة انكماش (Shrinking Phase) لإطلاقها عند انتهاء المعاملة.
2. **التحكم بالتزامن عبر تعدد الإصدارات (Multi-Version Concurrency Control - MVCC):**
   الأسلوب المعياري في قواعد البيانات الحديثة (PostgreSQL, MySQL InnoDB, Oracle). لا تعطل القراءة عمليات الكتابة، ولا تعطل الكتابة عمليات القراءة؛ حيث يرى كل استعلام لقطة متسقة (Consistent Snapshot) من البيانات محددة بطابع زمني أو رقم تسلسلي (Transaction ID / SCN).
3. **عزل اللقطات المتسلسل (Serializable Snapshot Isolation - SSI):**
   خوارزمية رائدة مطبقة في PostgreSQL تسمح بتنفيذ أعلى مستوى عزل (`Serializable`) بالاعتماد على MVCC مع تتبع مسارات التعارض (Conflict Graph Detection) وإلغاء المعاملات المتناقضة فقط، مما يحقق أداءً فائقاً مقارنة بالقفل التقليدي.

---

### د. الديمومة (Durability) — استمرارية البيانات وصمودها بعد التثبيت

#### المفهوم

تضمن الديمومة أنه بمجرد أن تتلقى المعاملة إشعار التثبيت الناجح (`Commit Acknowledgment`)، فإن التغييرات التي أحدثتها تصبح دائمة وغير قابلة للضياع، وتصمد أمام انقطاع الطاقة المفاجئ، أو انهيار نظام التشغيل (OS Crash)، أو تعطل العتاد الصلب (Hardware Failure).

#### آليات إدارة المخزن المؤقت (Buffer Management Taxonomy - Härder & Reuter)

صنّف هاردر ورويتر سياسات إدارة المخزن المؤقت للذاكرة والقرص إلى أربعة أنماط تتحكم في كيفية تحقيق الديمومة والذرية:

1. **Steal vs. No-Steal (إدارة الصفحات المعدلة):**
   * *Steal:* يُسمح لمدير الذاكرة بكتابة صفحات معدلة تخص معاملات لم تُثبّت بعد إلى القرص لإخلاء مساحة في الذاكرة (يتطلب وجود سجلات Undo للتراجع في حال فشل المعاملة).
   * *No-Steal:* لا تُكتب أي صفحة معدلة إلى القرص حتى تُثبت المعاملة بنجاح.
2. **Force vs. No-Force (إدارة التثبيت):**
   * *Force:* عند تنفيذ الـ Commit، يجب كتابة كافة صفحات البيانات المعدلة فوراً إلى القرص (يضمن الديمومة مباشرة لكنه يسبب اختناقاً كارثياً في أداء القرص I/O).
   * *No-Force:* تكتمل المعاملة بمجرد كتابة سجل الـ WAL فقط إلى القرص، بينما تبقى صفحات البيانات في الذاكرة وتُكتب لاحقاً على مهل (Lazy Writing). ويتطلب هذا توفر سجلات Redo لإعادة تطبيق التغييرات إذا انهار النظام قبل كتابتها.

> **النموذج الصناعي السائد:** تستخدم معظم قواعد البيانات الحديثة (PostgreSQL, MySQL, Oracle, SQL Server) استراتيجية **(Steal / No-Force)** لأنها تقدم أعلى درجات الأداء وتوازنها بأمان عبر الـ Write-Ahead Log.

#### الآليات الهندسية الإضافية للديمومة

* **نداء النظام `fsync()`:** يضمن إفراغ البيانات المخزنة مؤقتاً في ذاكرة نظام التشغيل (OS Page Cache) وكتابتها مادياً في القرص التخزيني (SSD / NVMe).
* **التثبيت الجماعي (Group Commit):** دمج عمليات `fsync` لعدة معاملات متزامنة في عملية كتابة واحدة على القرص لزيادة معدل الإنتاجية (Throughput).
* **التوافق الموزع (Distributed Consensus):** في الأنظمة الموزعة، تُعد المعاملة دائمة بمجرد كتابتها في سجلات أغلبية العقد عبر خوارزميات التوافق مثل Raft أو Paxos.

---

## 3. معاملات ACID في قواعد البيانات الموزعة (Distributed ACID)

مع ظهور قواعد البيانات الموزعة عالمياً والجيل الحديث المعروف بـ **NewSQL**، لم تعد المعاملات تقتصر على جهاز واحد:

1. **بروتوكول الالتزام ثنائي المراحل (Two-Phase Commit - 2PC):**
   خوارزمية كلاسيكية لتنسيق العمليات بين عدة عقد مستقلة عبر مرحلتين:
   * **مرحلة التصويت / التحضير (Prepare Phase):** يسأل المنسق كل العقد عما إذا كانت جاهزة للتثبيت، وتقوم كل عقدة بحجز الموارد وكتابة WAL.
   * **مرحلة الاعتماد (Commit Phase):** إذا وافقت جميع العقد، يرسل المنسق أمر التثبيت النهائي للجميع؛ وإن رفضت أي عقدة، يرسل أمر التراجع (`Rollback`) للجميع.

2. **Google Spanner وواجهة TrueTime API:**
   قدمت شركة Google في ورقتها البحثية (OSDI 2012) طفرة نوعية في دعم ACID موزع على مستوى العالم مع توفير اتساق خطي خارجي صارم (**External Consistency / Strict Serializability**).
   اعتمدت Spanner على دمج خوارزمية Paxos مع تقنية **TrueTime API** المعتمدة على ساعات ذرية (Atomic Clocks) ومستقبلات GPS متزامنة بدقة داخل مراكز البيانات للتغلب على عدم اليقين الزمني في الشبكات الموزعة.

---

## 4. مقارنة معمارية: نموذج ACID مقابل نموذج BASE

| وجه المقارنة | نموذج ACID (قواعد البيانات العلائقية RDBMS) | نموذج BASE (أنظمة NoSQL الموزعة) |
| :--- | :--- | :--- |
| **المعنى والمبادئ** | Atomicity, Consistency, Isolation, Durability | Basically Available, Soft state, Eventual consistency |
| **الفلسفة الجوهرية** | التشديد على صحة البيانات والاتساق الفوري الصارم | تفضيل التوافرية العالية وقابلية التوسع الأفقي الضخم |
| **طبيعة الاتساق** | اتساق فوري وقوي (Strong / Immediate Consistency) | اتساق نهائي في وقت لاحق (Eventual Consistency) |
| **التحكم بالتزامن** | إقفال متزامن، لقطات MVCC، عزل Serializable | آليات خالية من الأقفال (Lock-free)، تعارض القراءات المتأخرة |
| **أمثلة على الأنظمة** | PostgreSQL, MySQL, Oracle, CockroachDB | Apache Cassandra, DynamoDB, MongoDB, Couchbase |
| **أبرز حالات الاستخدام** | الأنظمة المالية والمصرفية، الدفع الإلكتروني، إدارة المخزون | الشبكات الاجتماعية، تحليلات البيانات الضخمة، أنظمة المراسلة الفورية |

---

## 5. المصادر والمراجع الرسمية المعتمدة (Authoritative Sources & Citations)

### الأوراق الأكاديمية والبحثية التأسيسية (Foundational Papers)

1. **Gray, Jim (1981):**
   * *The Transaction Concept: Virtues and Limitations.* Tandem Computers Technical Report / IEEE.
   * [ACM Digital Library - DOI: 10.5555/850626.850663](https://dl.acm.org/doi/10.5555/850626.850663)
2. **Härder, Theo & Reuter, Andreas (1983):**
   * *Principles of Transaction-Oriented Database Recovery.* ACM Computing Surveys (CSUR), Vol. 15, No. 4, pp. 287–317.
   * [ACM Digital Library - DOI: 10.1145/289.291](https://dl.acm.org/doi/10.1145/289.291)
3. **Mohan, C., Haderle, D., Lindsay, B., Pirahesh, H., & Schwarz, P. (1992):**
   * *ARIES: A Transaction Recovery Method Supporting Fine-Granularity Locking and Partial Rollbacks Using Write-Ahead Logging.* ACM Transactions on Database Systems (TODS), Vol. 17, No. 1, pp. 94–162.
   * [ACM Digital Library - DOI: 10.1145/128765.128770](https://dl.acm.org/doi/10.1145/128765.128770)
4. **Berenson, H., Bernstein, P., Gray, J., Melton, J., O'Neil, E., & O'Neil, P. (1995):**
   * *A Critique of ANSI SQL Isolation Levels.* ACM SIGMOD Record, Vol. 24, Issue 2, pp. 1–10.
   * [Microsoft Research / ACM Publication](https://www.microsoft.com/en-us/research/publication/a-critique-of-ansi-sql-isolation-levels/)
5. **Corbett, James C., et al. (Google Research, 2012):**
   * *Spanner: Google’s Globally-Distributed Database.* Proceedings of the 10th USENIX Conference on Operating Systems Design and Implementation (OSDI '12), pp. 251–264.
   * [USENIX Publication Link](https://www.usenix.org/conference/osdi12/technical-sessions/presentation/corbett) / [Google Research Link](https://research.google/pubs/pub39966/)

### التوثيق الرسمي لمحركات قواعد البيانات القياسية

1. **PostgreSQL Official Documentation:**
   * Chapter 13: *Concurrency Control & Multi-Version Concurrency Control (MVCC)*: [postgresql.org/docs/current/mvcc.html](https://www.postgresql.org/docs/current/mvcc.html)
   * Chapter 30: *Reliability and the Write-Ahead Log (WAL)*: [postgresql.org/docs/current/wal.html](https://www.postgresql.org/docs/current/wal.html)
2. **MySQL / InnoDB Reference Manual:**
   * Section 17.5.9: *InnoDB and the ACID Model*: [dev.mysql.com/doc/refman/en/mysql-acid.html](https://dev.mysql.com/doc/refman/en/mysql-acid.html)
   * Section 17.5.2: *InnoDB Multi-Versioning*: [dev.mysql.com/doc/refman/en/innodb-multi-versioning.html](https://dev.mysql.com/doc/refman/en/innodb-multi-versioning.html)
3. **Oracle Database Concepts Guide:**
   * Chapter 9: *Data Concurrency and Consistency*: [docs.oracle.com/en/database/oracle/oracle-database/19/cncpt/data-concurrency-and-consistency.html](https://docs.oracle.com/en/database/oracle/oracle-database/19/cncpt/data-concurrency-and-consistency.html)
4. **المعيار الدولي لـ SQL (ISO/IEC Standard):**
   * *ISO/IEC 9075-2: Information technology — Database languages — SQL — Part 2: Foundation (SQL/Foundation).* Transaction management and transaction isolation levels.
