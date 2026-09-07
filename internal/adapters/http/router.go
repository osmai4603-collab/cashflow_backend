package httpadapter

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	accountinghttp "cashflow_backend/internal/adapters/http/accounting"
	analytichttp "cashflow_backend/internal/adapters/http/analytic"
	attachmenthttp "cashflow_backend/internal/adapters/http/attachment"
	bankstatementhttp "cashflow_backend/internal/adapters/http/bankstatement"
	companyhttp "cashflow_backend/internal/adapters/http/company"
	crmhttp "cashflow_backend/internal/adapters/http/crm"
	currencyhttp "cashflow_backend/internal/adapters/http/currency"
	hrhttp "cashflow_backend/internal/adapters/http/hr"
	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	paymenthttp "cashflow_backend/internal/adapters/http/payment"
	producthttp "cashflow_backend/internal/adapters/http/product"
	projecthttp "cashflow_backend/internal/adapters/http/project"
	purchasehttp "cashflow_backend/internal/adapters/http/purchase"
	salehttp "cashflow_backend/internal/adapters/http/sale"
	sequencehttp "cashflow_backend/internal/adapters/http/sequence"
	stockhttp "cashflow_backend/internal/adapters/http/stock"
	userhttp "cashflow_backend/internal/adapters/http/user"
	activityhttp "cashflow_backend/internal/adapters/http/activity"
	mrphttp "cashflow_backend/internal/adapters/http/mrp"
	"cashflow_backend/internal/platform/auth"
	platconfig "cashflow_backend/internal/platform/config"
	"cashflow_backend/internal/platform/response"
)

// HealthRoutes defines the liveness and readiness probe endpoints.
type HealthRoutes interface {
	HandleLiveness(w http.ResponseWriter, r *http.Request)
	HandleReadiness(w http.ResponseWriter, r *http.Request)
}

// NewRouter initializes and configures the HTTP router with standard middlewares.
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
	r := chi.NewRouter()
	secret := "odoo-go-insecure-dev-secret-key-change-in-production"
	var authorizer auth.Authorizer
	var projectHandler *projecthttp.Handler
	var bankstatementHandler *bankstatementhttp.Handler
	var mrpHandler *mrphttp.Handler
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
		case *projecthttp.Handler:
			projectHandler = value
		case *bankstatementhttp.Handler:
			bankstatementHandler = value
		case *mrphttp.Handler:
			mrpHandler = value
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
	r.Get("/", handler.Root)

	if userHandler != nil {
		r.Post("/api/v1/users/login", userHandler.Login)
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
		if companyHandler != nil {
			companyhttp.RegisterRoutes(v1, companyHandler, authorizer)
		}
		if userHandler != nil {
			userhttp.RegisterRoutes(v1, userHandler)
		}
		if currencyHandler != nil {
			currencyhttp.RegisterRoutes(v1, currencyHandler)
		}
		if sequenceHandler != nil {
			sequencehttp.RegisterRoutes(v1, sequenceHandler)
		}
		if attachmentHandler != nil {
			attachmenthttp.RegisterRoutes(v1, attachmentHandler)
		}
		if activityHandler != nil {
			activityhttp.RegisterRoutes(v1, activityHandler, authorizer)
		}
		if projectHandler != nil {
			projecthttp.RegisterRoutes(v1, projectHandler, authorizer)
		}
		if bankstatementHandler != nil {
			bankstatementhttp.RegisterRoutes(v1, bankstatementHandler, authorizer)
		}

		// Business Modules
		if partnerHandler != nil {
			partnerhttp.RegisterRoutes(v1, partnerHandler, authorizer)
		}
		if productHandler != nil {
			producthttp.RegisterRoutes(v1, productHandler, authorizer)
		}
		if accountingHandler != nil {
			accountinghttp.RegisterRoutes(v1, accountingHandler, authorizer)
		}
		if analyticHandler != nil {
			analytichttp.RegisterRoutes(v1, analyticHandler)
		}
		if saleHandler != nil {
			salehttp.RegisterRoutes(v1, saleHandler, authorizer)
		}
		if purchaseHandler != nil {
			purchasehttp.RegisterRoutes(v1, purchaseHandler, authorizer)
		}
		if stockHandler != nil {
			stockhttp.RegisterRoutes(v1, stockHandler, authorizer)
		}
		if crmHandler != nil {
			crmhttp.RegisterRoutes(v1, crmHandler, authorizer)
		}
		if paymentHandler != nil {
			paymenthttp.RegisterRoutes(v1, paymentHandler, authorizer)
		}
		if hrHandler != nil {
			hrhttp.RegisterRoutes(v1, hrHandler, authorizer)
		}
		if mrpHandler != nil {
			v1.Mount("/mrp", mrpHandler.Routes())
		}
	})

	return r
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
