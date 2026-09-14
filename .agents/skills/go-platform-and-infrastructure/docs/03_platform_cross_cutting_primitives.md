# Platform Cross-Cutting Primitives

Platform primitives are reusable, thread-safe engines that solve fundamental technical challenges across domains: auditing changes, distributing internal events, generating sequential document numbers, and executing high-precision currency conversions.

---

## 1. Audit Context & Entity Tracking (`platform/audit`)

In enterprise ERP systems, every mutation must track **who** performed the action and **when**:

```go
type Fields struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy *int64    `json:"created_by,omitempty"`
	UpdatedBy *int64    `json:"updated_by,omitempty"`
}

func NewFields(ctx context.Context) Fields {
	now := time.Now().UTC()
	uid := UserIDFromContext(ctx)
	return Fields{
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: uid,
		UpdatedBy: uid,
	}
}

func (f *Fields) Touch(ctx context.Context) {
	f.UpdatedAt = time.Now().UTC()
	if uid := UserIDFromContext(ctx); uid != nil {
		f.UpdatedBy = uid
	}
}
```

---

## 2. In-Process Notification & Event Bus (`platform/notificationbus`)

The notification bus allows decouple event producers (e.g. invoice created, payment received) from internal subscribers (e.g. WebSocket broadcasters, analytics counters) without external message broker overhead.

### Non-Blocking Distribution Principle:
A slow or unresponsive subscriber must **never** block the publisher:

```go
func (b *Bus) NotifyUser(ctx context.Context, userID int64, channel string, payload any) error {
	event := Event{Channel: channel, Payload: payload}
	b.mu.RLock()
	defer b.mu.RUnlock()

	for sub := range b.subscribers[userID] {
		select {
		case sub <- event:
			// Dispatched successfully
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Buffer is full! Drop or log warning rather than blocking execution.
		}
	}
	return nil
}
```

---

## 3. Atomic Sequence Generation (`platform/sequence`)

Enterprise documents (invoices, purchase orders, payments) require sequential, gapless, or human-readable identifiers formatted with date tokens (e.g. `INV/2026/03/00142`).

### Requirements:
1. **Concurrency Safety**: Use database row locks (`SELECT ... FOR UPDATE`) on the sequence definition row to guarantee atomic counter incrementation.
2. **Date Token Expansion**: Format tokens like `%(year)s`, `%(month)s`, `%(day)s` dynamically at the time of creation.
3. **Padding**: Maintain zero-padding (e.g., width 5: `00042`).

---

## 4. Multi-Currency Arithmetic (`platform/currency`)

Financial math is vulnerable to IEEE-754 floating-point inaccuracies when calculating tax, currency conversion, and line item sums.

### Conversion Formula:
$$\text{Amount}_{\text{Target}} = \text{Amount}_{\text{Source}} \times \frac{\text{Rate}_{\text{Target}}}{\text{Rate}_{\text{Source}}}$$

Where rates are stored relative to a single company Anchor Currency (e.g. USD = 1.0, SAR = 3.75). When rates are equal or currencies match, bypass conversion to eliminate precision drift.
