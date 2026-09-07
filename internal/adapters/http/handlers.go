package httpadapter

import (
	"log/slog"

	accountinghttp "cashflow_backend/internal/adapters/http/accounting"
	activityhttp "cashflow_backend/internal/adapters/http/activity"
	analytichttp "cashflow_backend/internal/adapters/http/analytic"
	attachmenthttp "cashflow_backend/internal/adapters/http/attachment"
	bankstatementhttp "cashflow_backend/internal/adapters/http/bankstatement"
	companyhttp "cashflow_backend/internal/adapters/http/company"
	crmhttp "cashflow_backend/internal/adapters/http/crm"
	currencyhttp "cashflow_backend/internal/adapters/http/currency"
	hrhttp "cashflow_backend/internal/adapters/http/hr"
	mrphttp "cashflow_backend/internal/adapters/http/mrp"
	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	paymenthttp "cashflow_backend/internal/adapters/http/payment"
	producthttp "cashflow_backend/internal/adapters/http/product"
	projecthttp "cashflow_backend/internal/adapters/http/project"
	purchasehttp "cashflow_backend/internal/adapters/http/purchase"
	salehttp "cashflow_backend/internal/adapters/http/sale"
	sequencehttp "cashflow_backend/internal/adapters/http/sequence"
	stockhttp "cashflow_backend/internal/adapters/http/stock"
	userhttp "cashflow_backend/internal/adapters/http/user"
	"cashflow_backend/internal/platform/notificationbus"
	usecase "cashflow_backend/internal/usecase"
)

// CashflowHandlers contains the HTTP handlers for all application modules.
type CashflowHandlers struct {
	Base          *BaseHandler
	Partner       *partnerhttp.Handler
	Product       *producthttp.Handler
	Accounting    *accountinghttp.Handler
	Analytic      *analytichttp.Handler
	Sale          *salehttp.Handler
	Purchase      *purchasehttp.Handler
	Stock         *stockhttp.Handler
	CRM           *crmhttp.Handler
	Payment       *paymenthttp.Handler
	HR            *hrhttp.Handler
	Company       *companyhttp.Handler
	User          *userhttp.Handler
	Currency      *currencyhttp.Handler
	Sequence      *sequencehttp.Handler
	Attachment    *attachmenthttp.Handler
	Activity      *activityhttp.Handler
	Project       *projecthttp.Handler
	BankStatement *bankstatementhttp.Handler
	MRP           *mrphttp.Handler
}

// NewHandlers creates an empty handler container for the composition root.
func NewHandlers(
	useCases *usecase.CashflowUseCases,
	serviceName string,
	version string,
	logger *slog.Logger,
	localNotificationBus *notificationbus.Bus,
) *CashflowHandlers {
	if logger == nil {
		logger = slog.Default()
	}

	return &CashflowHandlers{
		Base:          NewBaseHandler(serviceName, version, logger),
		Partner:       partnerhttp.NewHandler(useCases.Partner, logger),
		Product:       producthttp.NewHandler(useCases.Product, logger),
		Accounting:    accountinghttp.NewHandler(useCases.Accounting, logger),
		Analytic:      analytichttp.NewHandler(useCases.Analytic, logger),
		Sale:          salehttp.NewHandler(useCases.Sale, logger),
		Purchase:      purchasehttp.NewHandler(useCases.Purchase, logger, useCases.Requisition),
		Stock:         stockhttp.NewHandler(useCases.Stock, logger),
		CRM:           crmhttp.NewHandler(useCases.CRM, logger),
		Payment:       paymenthttp.NewHandler(useCases.Payment, logger),
		HR:            hrhttp.NewHandler(useCases.HR, useCases.Attendance, logger),
		Company:       companyhttp.NewHandler(useCases.Company, logger),
		User:          userhttp.NewHandler(useCases.User, logger),
		Currency:      currencyhttp.NewHandler(useCases.Currency, logger),
		Sequence:      sequencehttp.NewHandler(useCases.Sequence, logger),
		Attachment:    attachmenthttp.NewHandler(useCases.Attachment, logger),
		Activity:      activityhttp.NewHandler(useCases.Activity, logger, localNotificationBus),
		Project:       projecthttp.NewHandler(useCases.Project),
		BankStatement: bankstatementhttp.NewHandler(useCases.BankStatement, logger),
		MRP:           mrphttp.NewHandler(useCases.MRP, logger),
	}
}
