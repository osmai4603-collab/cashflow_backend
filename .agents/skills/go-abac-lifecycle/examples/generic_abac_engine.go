package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// ============================================================================
// 1. DOMAIN-AGNOSTIC ATTRIBUTE MODELING (NIST SP 800-162)
// ============================================================================

// Subject represents an authenticated caller (user, service account, or system actor).
type Subject struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id,omitempty"` // For multi-tenant company isolation
	Roles      []string       `json:"roles,omitempty"`
	Department string         `json:"department,omitempty"`
	Clearance  int            `json:"clearance,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// Resource represents the target entity being operated upon.
type Resource struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"` // e.g., "document", "account", "invoice"
	OwnerID     string         `json:"owner_id,omitempty"`
	TenantID    string         `json:"tenant_id,omitempty"` // For multi-tenant company isolation
	Department  string         `json:"department,omitempty"`
	Sensitivity int            `json:"sensitivity,omitempty"`
	Status      string         `json:"status,omitempty"`
	Attributes  map[string]any `json:"attributes,omitempty"`
}

// Action represents the verb or operation requested on the resource.
type Action struct {
	Verb   string `json:"verb"`             // e.g., "read", "create", "update", "delete", "approve"
	Method string `json:"method,omitempty"` // HTTP verb or RPC method name
}

// Environment represents ambient contextual metadata at evaluation time.
type Environment struct {
	RequestTime time.Time      `json:"request_time"`
	ClientIP    string         `json:"client_ip,omitempty"`
	NetworkZone string         `json:"network_zone,omitempty"`
	Attributes  map[string]any `json:"attributes,omitempty"`
}

// EvaluationContext binds the complete 4-dimensional attribute quadruple.
type EvaluationContext struct {
	Subject     Subject     `json:"subject"`
	Resource    Resource    `json:"resource"`
	Action      Action      `json:"action"`
	Environment Environment `json:"environment"`
}

// Safe attribute extractors to prevent runtime panics on dynamic attributes
func GetStringAttr(attrs map[string]any, key, defaultVal string) string {
	if attrs == nil {
		return defaultVal
	}
	val, ok := attrs[key]
	if !ok {
		return defaultVal
	}
	if s, ok := val.(string); ok {
		return s
	}
	return defaultVal
}

func GetFloatAttr(attrs map[string]any, key string, defaultVal float64) float64 {
	if attrs == nil {
		return defaultVal
	}
	val, ok := attrs[key]
	if !ok {
		return defaultVal
	}
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return defaultVal
	}
}

func GetBoolAttr(attrs map[string]any, key string, defaultVal bool) bool {
	if attrs == nil {
		return defaultVal
	}
	val, ok := attrs[key]
	if !ok {
		return defaultVal
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return defaultVal
}

// ============================================================================
// 2. DECISIONS, COMBINING ALGORITHMS, & POLICY INTERFACES (PAP / PDP)
// ============================================================================

// Decision represents the outcome of an authorization evaluation.
type Decision int

const (
	DecisionNotApplicable Decision = iota
	DecisionPermit
	DecisionDeny
)

func (d Decision) String() string {
	switch d {
	case DecisionPermit:
		return "Permit"
	case DecisionDeny:
		return "Deny"
	default:
		return "NotApplicable"
	}
}

// CombiningAlgorithm determines how multiple rule decisions are resolved into a final outcome.
type CombiningAlgorithm int

const (
	DenyOverrides CombiningAlgorithm = iota // Financial / High-Security Standard
	PermitOverrides                         // Open collaboration / Permissive Standard
	FirstApplicable                         // Ordered firewall / Sequential Standard
)

// PolicyRule defines the contract for an atomic authorization rule.
type PolicyRule interface {
	ID() string
	Description() string
	Target(ctx EvaluationContext) bool
	Evaluate(ctx EvaluationContext) (Decision, string)
}

// RuleFunc is an adapter type allowing standard closures to act as PolicyRules.
type RuleFunc struct {
	RuleID       string
	Desc         string
	TargetFunc   func(ctx EvaluationContext) bool
	EvaluateFunc func(ctx EvaluationContext) (Decision, string)
}

func (r RuleFunc) ID() string                                 { return r.RuleID }
func (r RuleFunc) Description() string                        { return r.Desc }
func (r RuleFunc) Target(ctx EvaluationContext) bool          { return r.TargetFunc(ctx) }
func (r RuleFunc) Evaluate(ctx EvaluationContext) (Decision, string) {
	return r.EvaluateFunc(ctx)
}

// ============================================================================
// 3. POLICY DECISION POINT (PDP ENGINE)
// ============================================================================

// Engine encapsulates thread-safe policy storage, combining logic, and audit logging.
type Engine struct {
	mu        sync.RWMutex
	algorithm CombiningAlgorithm
	rules     []PolicyRule
	logger    *slog.Logger
}

// NewEngine constructs a new thread-safe ABAC Policy Decision Point.
func NewEngine(algorithm CombiningAlgorithm, logger *slog.Logger, rules ...PolicyRule) *Engine {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return &Engine{
		algorithm: algorithm,
		rules:     rules,
		logger:    logger,
	}
}

// RegisterRule appends a new rule to the engine under a write lock.
func (e *Engine) RegisterRule(rule PolicyRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, rule)
}

// ReloadRules atomically swaps the active policy rule set with zero downtime.
func (e *Engine) ReloadRules(newRules []PolicyRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = make([]PolicyRule, len(newRules))
	copy(e.rules, newRules)
}

// SetAlgorithm updates the combining algorithm under a write lock.
func (e *Engine) SetAlgorithm(alg CombiningAlgorithm) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.algorithm = alg
}

// Evaluate evaluates the context against all applicable rules using the configured combining algorithm.
func (e *Engine) Evaluate(ctx EvaluationContext) (Decision, string) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	start := time.Now()
	var finalDecision Decision
	var finalReason string

	switch e.algorithm {
	case DenyOverrides:
		finalDecision, finalReason = e.evaluateDenyOverrides(ctx)
	case PermitOverrides:
		finalDecision, finalReason = e.evaluatePermitOverrides(ctx)
	case FirstApplicable:
		finalDecision, finalReason = e.evaluateFirstApplicable(ctx)
	default:
		finalDecision, finalReason = DecisionDeny, "default deny: unrecognized combining algorithm"
	}

	latency := time.Since(start)

	// Emit structured audit log
	logLevel := slog.LevelInfo
	if finalDecision == DecisionDeny {
		logLevel = slog.LevelWarn
	}

	e.logger.Log(nil, logLevel, "ABAC_EVALUATION_COMPLETED",
		slog.String("decision", finalDecision.String()),
		slog.String("reason", finalReason),
		slog.String("subject_id", ctx.Subject.ID),
		slog.String("tenant_id", ctx.Subject.TenantID),
		slog.String("resource_type", ctx.Resource.Type),
		slog.String("resource_id", ctx.Resource.ID),
		slog.String("action", ctx.Action.Verb),
		slog.Duration("latency_ns", latency),
	)

	return finalDecision, finalReason
}

// evaluateDenyOverrides enforces strict deny: any Deny overrides any Permit.
// If no rule explicitly permits, it results in Default Deny.
func (e *Engine) evaluateDenyOverrides(ctx EvaluationContext) (Decision, string) {
	hasPermit := false
	permitReason := ""

	for _, rule := range e.rules {
		if !rule.Target(ctx) {
			continue
		}
		decision, reason := rule.Evaluate(ctx)
		if decision == DecisionDeny {
			return DecisionDeny, fmt.Sprintf("denied by rule %q: %s", rule.ID(), reason)
		}
		if decision == DecisionPermit {
			hasPermit = true
			permitReason = fmt.Sprintf("permitted by rule %q: %s", rule.ID(), reason)
		}
	}

	if hasPermit {
		return DecisionPermit, permitReason
	}
	return DecisionDeny, "default deny: no applicable rule permitted the action"
}

// evaluatePermitOverrides allows access if any rule permits, regardless of denials.
func (e *Engine) evaluatePermitOverrides(ctx EvaluationContext) (Decision, string) {
	hasDeny := false
	denyReason := ""

	for _, rule := range e.rules {
		if !rule.Target(ctx) {
			continue
		}
		decision, reason := rule.Evaluate(ctx)
		if decision == DecisionPermit {
			return DecisionPermit, fmt.Sprintf("permitted by rule %q: %s", rule.ID(), reason)
		}
		if decision == DecisionDeny {
			hasDeny = true
			denyReason = fmt.Sprintf("denied by rule %q: %s", rule.ID(), reason)
		}
	}

	if hasDeny {
		return DecisionDeny, denyReason
	}
	return DecisionDeny, "default deny: no applicable rule permitted the action"
}

// evaluateFirstApplicable returns the decision of the first applicable rule.
func (e *Engine) evaluateFirstApplicable(ctx EvaluationContext) (Decision, string) {
	for _, rule := range e.rules {
		if !rule.Target(ctx) {
			continue
		}
		decision, reason := rule.Evaluate(ctx)
		if decision == DecisionPermit || decision == DecisionDeny {
			return decision, fmt.Sprintf("decided by first-applicable rule %q (%s): %s", rule.ID(), decision.String(), reason)
		}
	}
	return DecisionDeny, "default deny: no applicable rule produced a decision"
}

// ============================================================================
// 4. POLICY INFORMATION POINT (PIP RESOLVER)
// ============================================================================

// PIPResolver defines the interface for dynamically fetching resource attributes.
type PIPResolver interface {
	ResolveResource(ctx context.Context, resourceType, resourceID string) (Resource, error)
}

// InMemoryPIPResolver is a thread-safe in-memory PIP provider for testing and caching.
type InMemoryPIPResolver struct {
	mu        sync.RWMutex
	resources map[string]Resource
}

// NewInMemoryPIPResolver initializes a new in-memory PIP resolver.
func NewInMemoryPIPResolver() *InMemoryPIPResolver {
	return &InMemoryPIPResolver{
		resources: make(map[string]Resource),
	}
}

// Put adds or updates a resource in the PIP store.
func (p *InMemoryPIPResolver) Put(res Resource) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := fmt.Sprintf("%s:%s", res.Type, res.ID)
	p.resources[key] = res
}

// ResolveResource retrieves resource attributes by type and ID.
func (p *InMemoryPIPResolver) ResolveResource(_ context.Context, resourceType, resourceID string) (Resource, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", resourceType, resourceID)
	res, ok := p.resources[key]
	if !ok {
		return Resource{}, errors.New("resource not found in PIP")
	}
	return res, nil
}

// ============================================================================
// 5. POLICY ENFORCEMENT POINT (PEP MIDDLEWARE & RFC 7807)
// ============================================================================

// Unexported context key type to make collisions mathematically impossible
type contextKey struct{}

var (
	subjectKey     = contextKey{}
	environmentKey = contextKey{}
)

// WithSubject stores the Subject in the context.
func WithSubject(ctx context.Context, sub Subject) context.Context {
	return context.WithValue(ctx, subjectKey, sub)
}

// GetSubject extracts the Subject from the context.
func GetSubject(ctx context.Context) (Subject, bool) {
	sub, ok := ctx.Value(subjectKey).(Subject)
	return sub, ok
}

// WithEnvironment stores the Environment in the context.
func WithEnvironment(ctx context.Context, env Environment) context.Context {
	return context.WithValue(ctx, environmentKey, env)
}

// GetEnvironment extracts the Environment from the context.
func GetEnvironment(ctx context.Context) (Environment, bool) {
	env, ok := ctx.Value(environmentKey).(Environment)
	return env, ok
}

// ProblemDetails represents an RFC 7807 error payload.
type ProblemDetails struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
}

// WriteForbiddenProblem returns a sanitized RFC 7807 response preventing information disclosure.
func WriteForbiddenProblem(w http.ResponseWriter, r *http.Request, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(ProblemDetails{
		Type:     "https://example.com/errors/forbidden",
		Title:    "Forbidden",
		Status:   http.StatusForbidden,
		Detail:   detail,
		Instance: r.URL.Path,
	})
}

// ResourceIDExtractor extracts resource type and resource ID from an HTTP request.
type ResourceIDExtractor func(r *http.Request) (resourceType string, resourceID string, err error)

// RequireABAC creates a standard PEP HTTP middleware guard.
func RequireABAC(engine *Engine, verb string, pip PIPResolver, extractor ResourceIDExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Extract Subject from context (populated by upstream AuthN middleware)
			subject, ok := GetSubject(r.Context())
			if !ok {
				WriteForbiddenProblem(w, r, "Access denied: unauthenticated subject context")
				return
			}

			// 2. Extract resource coordinates from request
			resType, resID, err := extractor(r)
			if err != nil {
				WriteForbiddenProblem(w, r, "Access denied: invalid resource identifier")
				return
			}

			// 3. PIP Dynamic Enrichment
			resource, err := pip.ResolveResource(r.Context(), resType, resID)
			if err != nil {
				WriteForbiddenProblem(w, r, "Access denied: unable to resolve resource")
				return
			}

			// 4. Resolve Environment
			env, ok := GetEnvironment(r.Context())
			if !ok {
				env = Environment{
					RequestTime: time.Now(),
					ClientIP:    r.RemoteAddr,
				}
			}

			// 5. Construct Quadruple
			evalCtx := EvaluationContext{
				Subject:     subject,
				Resource:    resource,
				Action:      Action{Verb: verb, Method: r.Method},
				Environment: env,
			}

			// 6. PDP Evaluation
			decision, _ := engine.Evaluate(evalCtx)
			if decision != DecisionPermit {
				// Prevent Information Disclosure: do not return internal rule names to clients
				WriteForbiddenProblem(w, r, "Access denied: you do not possess the required attributes to perform this action")
				return
			}

			// 7. Access Granted -> Forward to handler
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	fmt.Println("Generic ABAC Lifecycle Engine ready.")
}
