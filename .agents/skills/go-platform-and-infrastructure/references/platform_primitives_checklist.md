# Platform Primitives Quality & Architecture Checklist

Use this checklist during architecture reviews to verify that cross-cutting platform primitives are memory-safe, thread-safe, and decoupled from business logic.

---

## 1. Audit Logging (`platform/audit`)
- [ ] Are audit fields (`created_at`, `updated_at`, `created_by`, `updated_by`) embedded cleanly into domain entities?
- [ ] Are user IDs and tenant IDs retrieved securely from `context.Context` without type assertion panics?
- [ ] Does the audit event sink record who made the change, the action taken, and the timestamp in UTC?
- [ ] Is audit logging non-blocking or performed asynchronously when high throughput is required?

---

## 2. Notification & Event Bus (`platform/notificationbus`)
- [ ] Is the bus thread-safe (`sync.RWMutex`) across multiple subscriber registrations and dispatches?
- [ ] Do subscriber channels have appropriate buffer sizes (e.g. 16 or 32)?
- [ ] Does dispatch handle full subscriber channels gracefully (`select` with `default`) without blocking other clients?
- [ ] Does the `unsubscribe` closure cleanly remove subscriber channels and close them without double-closing?

---

## 3. Sequence Numbering (`platform/sequence`)
- [ ] Does the number source lock the sequence record using `SELECT ... FOR UPDATE`?
- [ ] Does the date token replacer expand `%(year)s`, `%(month)s`, and `%(day)s` correctly?
- [ ] Is zero-padding formatted reliably (e.g., `%05d`)?
- [ ] Are sequence numbers tested under concurrent load to verify absence of duplicate IDs?

---

## 4. Currency Math (`platform/currency`)
- [ ] Does currency conversion avoid precision loss (no direct floating point multiplications on raw amounts)?
- [ ] Is identity conversion (`fromCurrency == toCurrency`) short-circuited immediately?
- [ ] Are exchange rates resolved using historical dates when available, with explicit fallbacks?
- [ ] Are negative or zero rates rejected with validation errors?

---

## 5. Composition Root (`platform/app`)
- [ ] Are all dependencies instantiated explicitly in topological order?
- [ ] Is the composition root free from global mutable state?
- [ ] Are all resources (database connections, worker channels, servers) closed in reverse order on shutdown?
