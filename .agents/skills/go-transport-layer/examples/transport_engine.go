package transportexamples

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// 1. DOMAIN & TRANSPORT DATA CONTRACTS
// ============================================================================

// TransportProtocol identifies the network ingress channel.
type TransportProtocol string

const (
	ProtocolHTTP TransportProtocol = "HTTP"
	ProtocolGRPC TransportProtocol = "GRPC"
)

func (p TransportProtocol) String() string {
	return string(p)
}

// SecurityPrincipal represents an authenticated caller identity.
type SecurityPrincipal struct {
	UserID    string   `json:"user_id"`
	TenantID  string   `json:"tenant_id,omitempty"`
	Roles     []string `json:"roles,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	Subject   string   `json:"subject,omitempty"`
	IsAdmin   bool     `json:"is_admin"`
}

// TransportInfo holds diagnostic transport metadata injected into context.Context.
type TransportInfo struct {
	Protocol  TransportProtocol `json:"protocol"`
	RequestID string            `json:"request_id"`
	ClientIP  string            `json:"client_ip"`
	UserAgent string            `json:"user_agent"`
}

// Standard Domain Errors
var (
	ErrNotFound           = errors.New("resource not found")
	ErrUnauthorized       = errors.New("unauthorized: missing or invalid credentials")
	ErrForbidden          = errors.New("forbidden: insufficient privileges")
	ErrInvalidInput       = errors.New("invalid input data")
	ErrConflict           = errors.New("resource conflict")
	ErrPreconditionFailed = errors.New("precondition failed")
	ErrRateLimited        = errors.New("rate limit exceeded")
	ErrDeadlineExceeded   = errors.New("operation deadline exceeded")
	ErrInternal           = errors.New("internal server error")
)

// ============================================================================
// 2. UNEXPORTED TYPE-SAFE CONTEXT PROPAGATION
// ============================================================================

type contextKey int

const (
	transportInfoKey contextKey = iota
	principalKey
)

func WithTransportInfo(ctx context.Context, info TransportInfo) context.Context {
	return context.WithValue(ctx, transportInfoKey, info)
}

func GetTransportInfo(ctx context.Context) (TransportInfo, bool) {
	info, ok := ctx.Value(transportInfoKey).(TransportInfo)
	return info, ok
}

func WithPrincipal(ctx context.Context, p *SecurityPrincipal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

func GetPrincipal(ctx context.Context) (*SecurityPrincipal, bool) {
	p, ok := ctx.Value(principalKey).(*SecurityPrincipal)
	return p, ok && p != nil
}

// ============================================================================
// 3. TRANSPORT CONTEXT ABSTRACTION
// ============================================================================

type TransportContext interface {
	Protocol() TransportProtocol
	Context() context.Context
	RequestID() string
	ClientIP() string
	UserAgent() string
	Header(key string) string
	Principal() (*SecurityPrincipal, bool)
	SetPrincipal(principal *SecurityPrincipal)
}

// HTTP Implementation
type httpTransportContext struct {
	req       *http.Request
	requestID string
	principal *SecurityPrincipal
}

func NewHTTPTransportContext(req *http.Request, requestID string) TransportContext {
	return &httpTransportContext{
		req:       req,
		requestID: requestID,
	}
}

func (h *httpTransportContext) Protocol() TransportProtocol {
	return ProtocolHTTP
}

func (h *httpTransportContext) Context() context.Context {
	return h.req.Context()
}

func (h *httpTransportContext) RequestID() string {
	return h.requestID
}

func (h *httpTransportContext) ClientIP() string {
	if xff := h.req.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := h.req.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(h.req.RemoteAddr)
	if err != nil {
		return h.req.RemoteAddr
	}
	return host
}

func (h *httpTransportContext) UserAgent() string {
	return h.req.UserAgent()
}

func (h *httpTransportContext) Header(key string) string {
	return h.req.Header.Get(key)
}

func (h *httpTransportContext) Principal() (*SecurityPrincipal, bool) {
	if h.principal != nil {
		return h.principal, true
	}
	return GetPrincipal(h.req.Context())
}

func (h *httpTransportContext) SetPrincipal(principal *SecurityPrincipal) {
	h.principal = principal
	*h.req = *h.req.WithContext(WithPrincipal(h.req.Context(), principal))
}

// gRPC Implementation (Compatible with metadata.MD)
type GRPCPeer struct {
	Addr string
}

type GRPCTransportContext struct {
	ctx       context.Context
	metadata  map[string][]string
	peerAddr  string
	requestID string
	principal *SecurityPrincipal
}

func NewGRPCTransportContext(ctx context.Context, md map[string][]string, peerAddr, requestID string) TransportContext {
	if md == nil {
		md = make(map[string][]string)
	}
	return &GRPCTransportContext{
		ctx:       ctx,
		metadata:  md,
		peerAddr:  peerAddr,
		requestID: requestID,
	}
}

func (g *GRPCTransportContext) Protocol() TransportProtocol {
	return ProtocolGRPC
}

func (g *GRPCTransportContext) Context() context.Context {
	return g.ctx
}

func (g *GRPCTransportContext) RequestID() string {
	return g.requestID
}

func (g *GRPCTransportContext) ClientIP() string {
	if vals, ok := g.metadata["x-forwarded-for"]; ok && len(vals) > 0 {
		parts := strings.Split(vals[0], ",")
		return strings.TrimSpace(parts[0])
	}
	if vals, ok := g.metadata["x-real-ip"]; ok && len(vals) > 0 {
		return strings.TrimSpace(vals[0])
	}
	if g.peerAddr != "" {
		host, _, err := net.SplitHostPort(g.peerAddr)
		if err == nil {
			return host
		}
		return g.peerAddr
	}
	return "unknown"
}

func (g *GRPCTransportContext) UserAgent() string {
	if vals, ok := g.metadata["user-agent"]; ok && len(vals) > 0 {
		return vals[0]
	}
	return "grpc-client"
}

func (g *GRPCTransportContext) Header(key string) string {
	if vals, ok := g.metadata[strings.ToLower(key)]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (g *GRPCTransportContext) Principal() (*SecurityPrincipal, bool) {
	if g.principal != nil {
		return g.principal, true
	}
	return GetPrincipal(g.ctx)
}

func (g *GRPCTransportContext) SetPrincipal(principal *SecurityPrincipal) {
	g.principal = principal
	g.ctx = WithPrincipal(g.ctx, principal)
}

// ============================================================================
// 4. PRE-HANDLING PIPELINE ENGINE
// ============================================================================

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*SecurityPrincipal, error)
}

type PreHandlingPipeline struct {
	validator TokenValidator
	logger    *slog.Logger
	timeout   time.Duration
}

func NewPreHandlingPipeline(validator TokenValidator, logger *slog.Logger, timeout time.Duration) *PreHandlingPipeline {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &PreHandlingPipeline{
		validator: validator,
		logger:    logger,
		timeout:   timeout,
	}
}

func (p *PreHandlingPipeline) Execute(tc TransportContext) (context.Context, error) {
	start := time.Now()
	protocol := tc.Protocol()
	reqID := tc.RequestID()
	clientIP := tc.ClientIP()

	// Stage 1 & 2: Ingress logging & Correlation
	p.logger.Info("Ingress request received",
		slog.String("protocol", protocol.String()),
		slog.String("request_id", reqID),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", tc.UserAgent()),
	)

	// Stage 3: Context normalization
	ctx := tc.Context()
	ctx = WithTransportInfo(ctx, TransportInfo{
		Protocol:  protocol,
		RequestID: reqID,
		ClientIP:  clientIP,
		UserAgent: tc.UserAgent(),
	})

	// Stage 4: Authentication & Principal extraction
	authHeader := tc.Header("Authorization")
	if authHeader != "" {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)
		if token != "" && p.validator != nil {
			principal, err := p.validator.ValidateToken(ctx, token)
			if err != nil {
				p.logger.Warn("Authentication rejected",
					slog.String("protocol", protocol.String()),
					slog.String("request_id", reqID),
					slog.Any("error", err),
				)
				return nil, fmt.Errorf("%w: %s", ErrUnauthorized, err.Error())
			}
			tc.SetPrincipal(principal)
			ctx = WithPrincipal(ctx, principal)
		}
	}

	// Stage 6: Timeout enforcement
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && p.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		_ = cancel
	}

	p.logger.Debug("Pre-handling pipeline finished",
		slog.String("protocol", protocol.String()),
		slog.Duration("duration_us", time.Since(start)),
	)

	return ctx, nil
}

func GenerateCorrelationID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ============================================================================
// 5. PROTOCOL INGRESS ADAPTERS
// ============================================================================

// HTTP Middleware Adapter
func HTTPMiddleware(pipeline *PreHandlingPipeline) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Panic Recovery at Transport Edge
			defer func() {
				if rec := recover(); rec != nil {
					pipeline.logger.Error("Panic recovered at HTTP transport boundary",
						slog.Any("panic", rec),
					)
					WriteHTTPError(w, "", ErrInternal)
				}
			}()

			reqID := r.Header.Get("X-Request-ID")
			if reqID == "" {
				reqID = GenerateCorrelationID()
			}
			w.Header().Set("X-Request-ID", reqID)

			tc := NewHTTPTransportContext(r, reqID)
			enrichedCtx, err := pipeline.Execute(tc)
			if err != nil {
				WriteHTTPError(w, reqID, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(enrichedCtx))
		})
	}
}

// gRPC Interceptor Types (Standard gRPC-compatible signatures)
type UnaryServerInfo struct {
	FullMethod string
}

type UnaryHandler func(ctx context.Context, req any) (any, error)
type UnaryServerInterceptor func(ctx context.Context, req any, info *UnaryServerInfo, handler UnaryHandler) (any, error)

func GRPCUnaryInterceptor(pipeline *PreHandlingPipeline, outgoingMDSetter func(ctx context.Context, key, val string)) UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *UnaryServerInfo,
		handler UnaryHandler,
	) (resp any, err error) {
		// Panic Recovery at gRPC Transport Edge
		defer func() {
			if rec := recover(); rec != nil {
				pipeline.logger.Error("Panic recovered at gRPC transport boundary",
					slog.Any("panic", rec),
				)
				err = MapToGRPCError(ErrInternal)
			}
		}()

		reqID := GenerateCorrelationID()
		if outgoingMDSetter != nil {
			outgoingMDSetter(ctx, "x-request-id", reqID)
		}

		// In actual gRPC, incoming metadata is extracted via metadata.FromIncomingContext
		tc := NewGRPCTransportContext(ctx, nil, "", reqID)
		enrichedCtx, pErr := pipeline.Execute(tc)
		if pErr != nil {
			return nil, MapToGRPCError(pErr)
		}

		return handler(enrichedCtx, req)
	}
}

// ============================================================================
// 6. ERROR TRANSLATION & STATUS MAPPING
// ============================================================================

type ProblemDetails struct {
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
}

func WriteHTTPError(w http.ResponseWriter, reqID string, err error) {
	status := MapToHTTPStatus(err)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	problem := ProblemDetails{
		Title:    http.StatusText(status),
		Status:   status,
		Detail:   err.Error(),
		Instance: reqID,
	}
	_ = json.NewEncoder(w).Encode(problem)
}

func MapToHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrPreconditionFailed):
		return http.StatusPreconditionFailed
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrDeadlineExceeded):
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// gRPC Status Code Representation (Mirroring google.golang.org/grpc/codes)
type GRPCCode uint32

const (
	CodeOK                 GRPCCode = 0
	CodeCanceled           GRPCCode = 1
	CodeUnknown            GRPCCode = 2
	CodeInvalidArgument    GRPCCode = 3
	CodeDeadlineExceeded   GRPCCode = 4
	CodeNotFound           GRPCCode = 5
	CodeAlreadyExists      GRPCCode = 6
	CodePermissionDenied   GRPCCode = 7
	CodeResourceExhausted  GRPCCode = 8
	CodeFailedPrecondition GRPCCode = 9
	CodeAborted            GRPCCode = 10
	CodeOutOfRange         GRPCCode = 11
	CodeUnimplemented      GRPCCode = 12
	CodeInternal           GRPCCode = 13
	CodeUnavailable        GRPCCode = 14
	CodeDataLoss           GRPCCode = 15
	CodeUnauthenticated    GRPCCode = 16
)

type GRPCStatusError struct {
	Code    GRPCCode
	Message string
}

func (e *GRPCStatusError) Error() string {
	return fmt.Sprintf("rpc error: code = %d desc = %s", e.Code, e.Message)
}

func MapToGRPCError(err error) error {
	code := CodeInternal
	switch {
	case errors.Is(err, ErrNotFound):
		code = CodeNotFound
	case errors.Is(err, ErrUnauthorized):
		code = CodeUnauthenticated
	case errors.Is(err, ErrForbidden):
		code = CodePermissionDenied
	case errors.Is(err, ErrInvalidInput):
		code = CodeInvalidArgument
	case errors.Is(err, ErrConflict):
		code = CodeAlreadyExists
	case errors.Is(err, ErrPreconditionFailed):
		code = CodeFailedPrecondition
	case errors.Is(err, ErrRateLimited):
		code = CodeResourceExhausted
	case errors.Is(err, ErrDeadlineExceeded):
		code = CodeDeadlineExceeded
	}
	return &GRPCStatusError{Code: code, Message: err.Error()}
}

// ============================================================================
// 7. PURE BUSINESS USE CASE (Zero Transport Leakage)
// ============================================================================

type CreatePaymentDTO struct {
	AccountID string  `json:"account_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

type PaymentResult struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	ProtocolUsed  string `json:"protocol_used"`
	CallerUserID  string `json:"caller_user_id"`
}

type PaymentUseCase interface {
	ProcessPayment(ctx context.Context, req CreatePaymentDTO) (*PaymentResult, error)
}

type PaymentService struct {
	mu           sync.Mutex
	transactions map[string]*PaymentResult
}

func NewPaymentService() *PaymentService {
	return &PaymentService{
		transactions: make(map[string]*PaymentResult),
	}
}

func (s *PaymentService) ProcessPayment(ctx context.Context, req CreatePaymentDTO) (*PaymentResult, error) {
	// 1. Syntactic / Semantic Validation
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be strictly positive", ErrInvalidInput)
	}
	if req.Currency == "" {
		return nil, fmt.Errorf("%w: currency is required", ErrInvalidInput)
	}

	// 2. Caller Identity Verification from Context
	principal, ok := GetPrincipal(ctx)
	if !ok {
		return nil, ErrUnauthorized
	}

	// 3. Protocol Discovery (Self-Aware Telemetry)
	tInfo, _ := GetTransportInfo(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()

	txID := "tx_" + GenerateCorrelationID()[:8]
	res := &PaymentResult{
		TransactionID: txID,
		Status:        "CONFIRMED",
		ProtocolUsed:  tInfo.Protocol.String(),
		CallerUserID:  principal.UserID,
	}
	s.transactions[txID] = res

	return res, nil
}
