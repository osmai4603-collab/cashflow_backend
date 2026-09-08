package httpadapter

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	accountinghttp "cashflow_backend/internal/adapters/http/accounting"
	activityhttp "cashflow_backend/internal/adapters/http/activity"
	analytichttp "cashflow_backend/internal/adapters/http/analytic"
	attachmenthttp "cashflow_backend/internal/adapters/http/attachment"
	bankstatementhttp "cashflow_backend/internal/adapters/http/bankstatement"
	companyhttp "cashflow_backend/internal/adapters/http/company"
	crmhttp "cashflow_backend/internal/adapters/http/crm"
	currencyhttp "cashflow_backend/internal/adapters/http/currency"
	deliveryhttp "cashflow_backend/internal/adapters/http/delivery"
	expensehttp "cashflow_backend/internal/adapters/http/expense"
	fleethttp "cashflow_backend/internal/adapters/http/fleet"
	hrhttp "cashflow_backend/internal/adapters/http/hr"
	maintenancehttp "cashflow_backend/internal/adapters/http/maintenance"
	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	paymenthttp "cashflow_backend/internal/adapters/http/payment"
	producthttp "cashflow_backend/internal/adapters/http/product"
	projecthttp "cashflow_backend/internal/adapters/http/project"
	purchasehttp "cashflow_backend/internal/adapters/http/purchase"
	salehttp "cashflow_backend/internal/adapters/http/sale"
	sequencehttp "cashflow_backend/internal/adapters/http/sequence"
	stockhttp "cashflow_backend/internal/adapters/http/stock"
	userhttp "cashflow_backend/internal/adapters/http/user"
	"cashflow_backend/internal/platform/auth"
	platconfig "cashflow_backend/internal/platform/config"
	"cashflow_backend/internal/platform/response"
)

// HealthRoutes defines the liveness and readiness probe endpoints.
type HealthRoutes interface {
	HandleLiveness(w http.ResponseWriter, r *http.Request)
	HandleReadiness(w http.ResponseWriter, r *http.Request)
}

// NewRouterWithHandlers initializes the HTTP router from a handler container.
func NewRouterWithHandlers(
	handlers *CashflowHandlers,
	health HealthRoutes,
	logger *slog.Logger,
	options ...any,
) chi.Router {
	r := chi.NewRouter()
	secret := "odoo-go-insecure-dev-secret-key-change-in-production"
	var authorizer auth.Authorizer
	proxyMode := true
	requestTimeout := 60 * time.Second
	for _, option := range options {
		switch value := option.(type) {
		case string:
			if value != "" {
				secret = value
			}
		case auth.Authorizer:
			authorizer = value
		case *platconfig.Configuration:
			if value != nil {
				if value.Auth.JWTSecret != "" {
					secret = value.Auth.JWTSecret
				}
				proxyMode = value.Server.ProxyMode
				if value.Limit.TimeReal > 0 {
					requestTimeout = value.Limit.TimeReal
				}
			}
		}
	}

	// Global Middlewares
	r.Use(middleware.RequestID)
	if proxyMode {
		// Reverse proxy mode: trust X-Forwarded-For / X-Real-IP (Odoo --proxy-mode).
		r.Use(middleware.RealIP)
	}
	r.Use(structuredLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(requestTimeout))

	// Health Endpoints (/livez and /readyz)
	if health != nil {
		r.Get("/livez", health.HandleLiveness)
		r.Get("/readyz", health.HandleReadiness)
	}

	// Root Endpoint
	if handlers == nil {
		handlers = &CashflowHandlers{}
	}
	if handlers.Base != nil {
		r.Get("/", handlers.Base.Root)
	}

	if handlers.User != nil {
		r.Post("/api/v1/users/login", handlers.User.Login)
	}

	// API v1 Mount Point
	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Use(auth.Middleware(secret))

		apiRoot := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response.JSON(w, http.StatusOK, map[string]string{
				"message": "ERP API v1 is online and ready for business modules",
			})
		})
		if authorizer != nil {
			v1.With(auth.RequireAccess(authorizer, "system.api", auth.ActionRead)).Get("/", apiRoot)
		} else {
			v1.Get("/", apiRoot)
		}

		// Core Infrastructure Modules
		if handlers.Company != nil {
			companyhttp.RegisterRoutes(v1, handlers.Company, authorizer)
		}
		if handlers.User != nil {
			userhttp.RegisterRoutes(v1, handlers.User)
		}
		if handlers.Currency != nil {
			currencyhttp.RegisterRoutes(v1, handlers.Currency)
		}
		if handlers.Sequence != nil {
			sequencehttp.RegisterRoutes(v1, handlers.Sequence)
		}
		if handlers.Attachment != nil {
			attachmenthttp.RegisterRoutes(v1, handlers.Attachment)
		}
		if handlers.Activity != nil {
			activityhttp.RegisterRoutes(v1, handlers.Activity, authorizer)
		}
		if handlers.Project != nil {
			projecthttp.RegisterRoutes(v1, handlers.Project, authorizer)
		}
		if handlers.BankStatement != nil {
			bankstatementhttp.RegisterRoutes(v1, handlers.BankStatement, authorizer)
		}

		// Business Modules
		if handlers.Partner != nil {
			partnerhttp.RegisterRoutes(v1, handlers.Partner, authorizer)
		}
		if handlers.Product != nil {
			producthttp.RegisterRoutes(v1, handlers.Product, authorizer)
		}
		if handlers.Accounting != nil {
			accountinghttp.RegisterRoutes(v1, handlers.Accounting, authorizer)
		}
		if handlers.Analytic != nil {
			analytichttp.RegisterRoutes(v1, handlers.Analytic)
		}
		if handlers.Sale != nil {
			salehttp.RegisterRoutes(v1, handlers.Sale, authorizer)
		}
		if handlers.Purchase != nil {
			purchasehttp.RegisterRoutes(v1, handlers.Purchase, authorizer)
		}
		if handlers.Stock != nil {
			stockhttp.RegisterRoutes(v1, handlers.Stock, authorizer)
		}
		if handlers.CRM != nil {
			crmhttp.RegisterRoutes(v1, handlers.CRM, authorizer)
		}
		if handlers.Expense != nil {
			expensehttp.RegisterRoutes(v1, handlers.Expense, authorizer)
		}
		if handlers.Payment != nil {
			paymenthttp.RegisterRoutes(v1, handlers.Payment, authorizer)
		}
		if handlers.HR != nil {
			hrhttp.RegisterRoutes(v1, handlers.HR, authorizer)
		}
		if handlers.MRP != nil {
			v1.Mount("/mrp", handlers.MRP.Routes())
		}
		if handlers.Loyalty != nil {
			v1.Mount("/loyalty", handlers.Loyalty.Routes())
		}
		if handlers.Maintenance != nil {
			maintenancehttp.RegisterRoutes(v1, handlers.Maintenance, authorizer)
		}
		if handlers.Fleet != nil {
			fleethttp.RegisterRoutes(v1, handlers.Fleet, authorizer)
		}
		if handlers.Delivery != nil {
			deliveryhttp.RegisterRoutes(v1, handlers.Delivery, authorizer)
		}
	})

	return r
}

// NewRouter preserves the original constructor for callers that still provide
// handlers individually.
func NewRouter(
	handler *BaseHandler,
	health HealthRoutes,
	partnerHandler *partnerhttp.Handler,
	productHandler *producthttp.Handler,
	accountingHandler *accountinghttp.Handler,
	analyticHandler *analytichttp.Handler,
	saleHandler *salehttp.Handler,
	purchaseHandler *purchasehttp.Handler,
	stockHandler *stockhttp.Handler,
	crmHandler *crmhttp.Handler,
	paymentHandler *paymenthttp.Handler,
	hrHandler *hrhttp.Handler,
	companyHandler *companyhttp.Handler,
	userHandler *userhttp.Handler,
	currencyHandler *currencyhttp.Handler,
	sequenceHandler *sequencehttp.Handler,
	attachmentHandler *attachmenthttp.Handler,
	activityHandler *activityhttp.Handler,
	logger *slog.Logger,
	options ...any,
) chi.Router {
	return NewRouterWithHandlers(&CashflowHandlers{
		Base:       handler,
		Partner:    partnerHandler,
		Product:    productHandler,
		Accounting: accountingHandler,
		Analytic:   analyticHandler,
		Sale:       saleHandler,
		Purchase:   purchaseHandler,
		Stock:      stockHandler,
		CRM:        crmHandler,
		Payment:    paymentHandler,
		HR:         hrHandler,
		Company:    companyHandler,
		User:       userHandler,
		Currency:   currencyHandler,
		Sequence:   sequenceHandler,
		Attachment: attachmentHandler,
		Activity:   activityHandler,
	}, health, logger, options...)
}

func structuredLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
