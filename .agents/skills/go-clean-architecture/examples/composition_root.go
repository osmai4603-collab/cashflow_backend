package cleanarch

import (
	"database/sql"
	"net/http"
)

// =============================================================================
// LAYER 5: PLATFORM LAYER (Composition Root & Dependency Injection)
// =============================================================================
// Rules:
// - The ONLY place where all layers meet and are assembled.
// - Explicit manual wiring: No magical reflection containers.
// - Returns a ready-to-serve http.Handler.
// =============================================================================

type Application struct {
	Router http.Handler
	DB     *sql.DB
}

// BuildApplication acts as the Composition Root for the service.
func BuildApplication(db *sql.DB) *Application {
	// 1. Initialize Driven (Secondary) Adapters
	invoiceRepo := NewPostgresInvoiceRepository(db)

	// 2. Initialize Use Cases (Injecting Ports)
	payInvoiceUseCase := NewPayInvoiceUseCase(invoiceRepo, nil)

	// 3. Initialize Driving (Primary) Adapters
	invoiceHandler := NewInvoiceHTTPHandler(payInvoiceUseCase)

	// 4. Wire Routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/invoices/pay", invoiceHandler.HandlePay)

	return &Application{
		Router: mux,
		DB:     db,
	}
}
