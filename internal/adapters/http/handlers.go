package httpadapter

import (
	"log/slog"

	accountinghttp "cashflow_backend/internal/adapters/http/accounting"
	activityhttp "cashflow_backend/internal/adapters/http/activity"
	analytichttp "cashflow_backend/internal/adapters/http/analytic"
	attachmenthttp "cashflow_backend/internal/adapters/http/attachment"
	bankstatementhttp "cashflow_backend/internal/adapters/http/bankstatement"
	calendarhttp "cashflow_backend/internal/adapters/http/calendar"
	companyhttp "cashflow_backend/internal/adapters/http/company"
	crmhttp "cashflow_backend/internal/adapters/http/crm"
	currencyhttp "cashflow_backend/internal/adapters/http/currency"
	databasehttp "cashflow_backend/internal/adapters/http/database"
	deliveryhttp "cashflow_backend/internal/adapters/http/delivery"
	expensehttp "cashflow_backend/internal/adapters/http/expense"
	fleethttp "cashflow_backend/internal/adapters/http/fleet"
	hrhttp "cashflow_backend/internal/adapters/http/hr"
	loyaltyhttp "cashflow_backend/internal/adapters/http/loyalty"
	maintenancehttp "cashflow_backend/internal/adapters/http/maintenance"
	mrphttp "cashflow_backend/internal/adapters/http/mrp"
	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	paymenthttp "cashflow_backend/internal/adapters/http/payment"
	poshttp "cashflow_backend/internal/adapters/http/pos"
	producthttp "cashflow_backend/internal/adapters/http/product"
	projecthttp "cashflow_backend/internal/adapters/http/project"
	purchasehttp "cashflow_backend/internal/adapters/http/purchase"
	recruitmenthttp "cashflow_backend/internal/adapters/http/recruitment"
	reporthttp "cashflow_backend/internal/adapters/http/report"
	resourcehttp "cashflow_backend/internal/adapters/http/resource"
	salehttp "cashflow_backend/internal/adapters/http/sale"
	sequencehttp "cashflow_backend/internal/adapters/http/sequence"
	stockhttp "cashflow_backend/internal/adapters/http/stock"
	timesheethttp "cashflow_backend/internal/adapters/http/timesheet"
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
	Expense       *expensehttp.Handler
	Payment       *paymenthttp.Handler
	Pos           *poshttp.Handler
	Recruitment   *recruitmenthttp.Handler
	Timesheet     *timesheethttp.Handler
	Resource      *resourcehttp.Handler
	HR            *hrhttp.Handler
	Company       *companyhttp.Handler
	User          *userhttp.Handler
	Currency      *currencyhttp.Handler
	Sequence      *sequencehttp.Handler
	Attachment    *attachmenthttp.Handler
	Activity      *activityhttp.Handler
	Project       *projecthttp.Handler
	BankStatement *bankstatementhttp.Handler
	Calendar      *calendarhttp.Handler
	MRP           *mrphttp.Handler
	Loyalty       *loyaltyhttp.Handler
	Maintenance   *maintenancehttp.Handler
	Fleet         *fleethttp.Handler
	Delivery      *deliveryhttp.Handler
	Report        *reporthttp.Handler
	Database      *databasehttp.Handler
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
		Expense:       expensehttp.NewHandler(useCases.Expense),
		Payment:       paymenthttp.NewHandler(useCases.Payment, logger),
		Pos:           poshttp.NewHandler(useCases.Pos),
		Recruitment:   recruitmenthttp.NewHandler(useCases.Recruitment),
		Timesheet:     timesheethttp.NewHandler(useCases.Timesheet),
		Resource:      resourcehttp.NewHandler(useCases.Resource),
		HR:            hrhttp.NewHandler(useCases.HR, useCases.Attendance, logger),
		Company:       companyhttp.NewHandler(useCases.Company, logger),
		User:          userhttp.NewHandler(useCases.User, logger),
		Currency:      currencyhttp.NewHandler(useCases.Currency, logger),
		Sequence:      sequencehttp.NewHandler(useCases.Sequence, logger),
		Attachment:    attachmenthttp.NewHandler(useCases.Attachment, logger),
		Activity:      activityhttp.NewHandler(useCases.Activity, logger, localNotificationBus),
		Project:       projecthttp.NewHandler(useCases.Project),
		BankStatement: bankstatementhttp.NewHandler(useCases.BankStatement, logger),
		Calendar:      calendarhttp.NewHandler(useCases.Calendar),
		MRP:           mrphttp.NewHandler(useCases.MRP, logger),
		Loyalty:       loyaltyhttp.NewHandler(useCases.Loyalty, logger),
		Maintenance:   maintenancehttp.NewHandler(useCases.Maintenance, logger),
		Fleet:         fleethttp.NewHandler(useCases.Fleet, logger),
		Delivery:      deliveryhttp.NewHandler(useCases.Delivery),
		Report:        reporthttp.NewHandler(useCases.ReportGenerator, useCases.Dashboard),
	}
}
