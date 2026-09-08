package usecase

import (
	"log/slog"
	"time"

	"cashflow_backend/internal/adapters/storage"
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/activity"
	bankstatementdomain "cashflow_backend/internal/domain/bankstatement"
	"cashflow_backend/internal/platform/auth"
	platformcurrency "cashflow_backend/internal/platform/currency"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	activityusecase "cashflow_backend/internal/usecase/activity"
	analyticusecase "cashflow_backend/internal/usecase/analytic"
	attachmentusecase "cashflow_backend/internal/usecase/attachment"
	bankstatementusecase "cashflow_backend/internal/usecase/bankstatement"
	companyusecase "cashflow_backend/internal/usecase/company"
	crmusecase "cashflow_backend/internal/usecase/crm"
	currencyusecase "cashflow_backend/internal/usecase/currency"
	deliveryusecase "cashflow_backend/internal/usecase/delivery"
	expenseusecase "cashflow_backend/internal/usecase/expense"
	fleetusecase "cashflow_backend/internal/usecase/fleet"
	hrusecase "cashflow_backend/internal/usecase/hr"
	loyaltyusecase "cashflow_backend/internal/usecase/loyalty"
	maintenanceusecase "cashflow_backend/internal/usecase/maintenance"
	mrpusecase "cashflow_backend/internal/usecase/mrp"
	partnerusecase "cashflow_backend/internal/usecase/partner"
	paymentusecase "cashflow_backend/internal/usecase/payment"
	productusecase "cashflow_backend/internal/usecase/product"
	projectusecase "cashflow_backend/internal/usecase/project"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"
	saleusecase "cashflow_backend/internal/usecase/sale"
	sequenceusecase "cashflow_backend/internal/usecase/sequence"
	stockusecase "cashflow_backend/internal/usecase/stock"
	userusecase "cashflow_backend/internal/usecase/user"
)

type CashflowUseCases struct {
	Partner       *partnerusecase.PartnerUseCase
	Product       *productusecase.ProductUseCase
	Accounting    *accountingusecase.UseCase
	Analytic      *analyticusecase.UseCase
	Sale          *saleusecase.UseCase
	Purchase      *purchaseusecase.UseCase
	Requisition   *purchaseusecase.RequisitionUseCase
	Stock         *stockusecase.UseCase
	CRM           *crmusecase.UseCase
	Expense       *expenseusecase.UseCase
	Payment       *paymentusecase.UseCase
	BankStatement *bankstatementusecase.UseCase
	Attendance    *hrusecase.AttendanceUseCase
	HR            *hrusecase.UseCase
	Company       *companyusecase.CompanyUseCase
	User          *userusecase.UserUseCase
	Currency      *currencyusecase.CurrencyUseCase
	Sequence      *sequenceusecase.SequenceUseCase
	Attachment    *attachmentusecase.AttachmentUseCase
	Activity      *activityusecase.UseCase
	Project       *projectusecase.Service
	MRP           *mrpusecase.Usecase
	Loyalty       *loyaltyusecase.UseCase
	Maintenance   *maintenanceusecase.Service
	Fleet         *fleetusecase.Service
	Delivery      *deliveryusecase.UseCase
}

func New(
	repositories *storage.CashflowRepositories,
	logger *slog.Logger,
	authorizer auth.Authorizer,
	jwtSecret string,
	tokenTTL time.Duration,
	activityBus activity.Bus,
	statementNameProvider bankstatementdomain.StatementSequenceProvider,
) *CashflowUseCases {
	if logger == nil {
		logger = slog.Default()
	}

	useCases := &CashflowUseCases{}
	useCases.Partner = partnerusecase.New(repositories.Partner, logger, authorizer)
	useCases.Product = productusecase.New(repositories.Product, logger)
	useCases.Accounting = accountingusecase.New(repositories.Accounting, logger)
	zatcaProc := accountingusecase.NewZatcaProcessor(repositories.Company, repositories.Partner)
	useCases.Accounting.RegisterEDIProcessor(accounting.EDIFormatZatcaPhase1, zatcaProc)
	useCases.Accounting.RegisterEDIProcessor(accounting.EDIFormatZatcaPhase2, zatcaProc)
	useCases.Analytic = analyticusecase.New(repositories.Analytic, logger)
	useCases.Activity = activityusecase.NewUseCase(repositories.Activity, repositories.ActivityType, repositories.ActivityMsg, repositories.ActivityNotif, repositories.EmailQueue, repositories.ActivityFollower, repositories.ActivitySubtype, activityBus, repositories.User)
	useCases.Loyalty = loyaltyusecase.New(repositories.Loyalty, repositories.Product, repositories.Sale, logger)
	useCases.Sale = saleusecase.New(repositories.Sale, repositories.Partner, repositories.Product, repositories.Accounting, useCases.Accounting, logger, useCases.Loyalty, useCases.Activity)
	useCases.Purchase = purchaseusecase.New(repositories.Purchase, repositories.Partner, repositories.Product, repositories.Accounting, useCases.Accounting, logger)
	useCases.Requisition = purchaseusecase.NewRequisitionUseCase(repositories.Requisition, repositories.Purchase, repositories.SupplierInfo)
	useCases.Stock = stockusecase.New(repositories.Stock, repositories.Partner, repositories.Product, repositories.Sale, repositories.Purchase, useCases.Accounting, repositories.Company, logger)
	useCases.CRM = crmusecase.New(repositories.CRM, useCases.Partner, useCases.Sale, logger)
	expenseAccounting := expenseusecase.NewAccountingIntegration(useCases.Accounting, repositories.HR)
	useCases.Expense = expenseusecase.New(repositories.Expense, repositories.HR, logger, expenseAccounting)
	useCases.Expense.ConfigureActivityScheduler(useCases.Activity, 4) // seeded To-Do activity type
	useCases.Payment = paymentusecase.New(repositories.Payment, useCases.Accounting, repositories.Partner, logger)
	useCases.BankStatement = bankstatementusecase.New(repositories.BankStatement, useCases.Accounting, statementNameProvider, logger)
	useCases.Attendance = hrusecase.NewAttendanceUseCase(repositories.HR, repositories.Company, logger)
	useCases.HR = hrusecase.New(repositories.HR, repositories.Partner, logger)
	useCases.Company = companyusecase.New(repositories.Company, logger)
	useCases.User = userusecase.New(repositories.User, repositories.Partner, logger, jwtSecret, tokenTTL)
	currencyConverter := platformcurrency.NewConverter(repositories.CurrencyRates)
	useCases.Currency = currencyusecase.New(repositories.Currency, repositories.CurrencyRates, currencyConverter, logger)
	useCases.Sequence = sequenceusecase.New(repositories.Sequence, logger)
	useCases.Attachment = attachmentusecase.New(repositories.Attachment, logger)
	useCases.Project = projectusecase.New(repositories.Project)
	useCases.MRP = mrpusecase.NewUsecase(repositories.MRP, useCases.Sequence, repositories.Stock, useCases.Accounting, repositories.Product)
	useCases.Maintenance = maintenanceusecase.New(repositories.Maintenance, useCases.Activity, logger)
	useCases.Fleet = fleetusecase.New(repositories.Fleet, useCases.Activity, logger)
	useCases.Delivery = deliveryusecase.NewUseCase(repositories.Delivery, repositories.Sale, repositories.Product, logger)
	return useCases
}
