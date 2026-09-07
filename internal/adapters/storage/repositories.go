package storage

import (
	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	activitystorage "cashflow_backend/internal/adapters/storage/activity"
	analyticstorage "cashflow_backend/internal/adapters/storage/analytic"
	attachmentstorage "cashflow_backend/internal/adapters/storage/attachment"
	bankstatementstorage "cashflow_backend/internal/adapters/storage/bankstatement"
	companystorage "cashflow_backend/internal/adapters/storage/company"
	crmstorage "cashflow_backend/internal/adapters/storage/crm"
	currencystorage "cashflow_backend/internal/adapters/storage/currency"
	groupstorage "cashflow_backend/internal/adapters/storage/group"
	hrstorage "cashflow_backend/internal/adapters/storage/hr"
	mrpstorage "cashflow_backend/internal/adapters/storage/mrp"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	paymentstorage "cashflow_backend/internal/adapters/storage/payment"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	projectstorage "cashflow_backend/internal/adapters/storage/project"
	purchasestorage "cashflow_backend/internal/adapters/storage/purchase"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	sequencestorage "cashflow_backend/internal/adapters/storage/sequence"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	userstorage "cashflow_backend/internal/adapters/storage/user"
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/domain/analytic"
	"cashflow_backend/internal/domain/attachment"
	"cashflow_backend/internal/domain/bankstatement"
	"cashflow_backend/internal/domain/company"
	"cashflow_backend/internal/domain/crm"
	"cashflow_backend/internal/domain/currency"
	"cashflow_backend/internal/domain/group"
	"cashflow_backend/internal/domain/hr"
	"cashflow_backend/internal/domain/mrp"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/payment"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/project"
	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/domain/sequence"
	"cashflow_backend/internal/domain/stock"
	"cashflow_backend/internal/domain/user"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CashflowRepositories struct {
	Partner       partner.Repository
	Product       product.Repository
	Accounting    accounting.Repository
	Analytic      analytic.Repository
	Sale          sale.Repository
	Purchase      purchase.Repository
	Stock         stock.Repository
	CRM           crm.Repository
	Payment       payment.Repository
	HR            hr.Repository
	Company       company.Repository
	User          user.Repository
	Currency      currency.Repository
	CurrencyRates currency.RateRepository
	Sequence      sequence.Repository
	Attachment    attachment.Repository
	BankStatement bankstatement.Repository
	Project       project.Repository
	Permission    group.PermissionRepository
	MRP           mrp.Repository
	Requisition   purchaseusecase.RequisitionRepository
	Activity      activity.ActivityRepository
	ActivityType  activity.ActivityTypeRepository
	ActivityMsg   activity.MessageRepository
	ActivityNotif activity.NotificationRepository
	EmailQueue    activity.EmailQueueRepository
}

func NewFromPostgres(pool *pgxpool.Pool) *CashflowRepositories {
	activityRepo := activitystorage.NewPostgresRepo(pool)
	return &CashflowRepositories{
		Partner:       partnerstorage.NewPostgresRepo(pool),
		Product:       productstorage.NewPostgresRepo(pool),
		Accounting:    accountingstorage.NewPostgresRepo(pool),
		Analytic:      analyticstorage.NewPostgresRepo(pool),
		Sale:          salestorage.NewPostgresRepo(pool),
		Purchase:      purchasestorage.NewPostgresRepo(pool),
		Stock:         stockstorage.NewPostgresRepo(pool),
		CRM:           crmstorage.NewPostgresRepo(pool),
		Payment:       paymentstorage.NewPostgresRepo(pool),
		HR:            hrstorage.NewPostgresRepo(pool),
		Company:       companystorage.NewPostgresRepo(pool),
		User:          userstorage.NewPostgresRepo(pool),
		Currency:      currencystorage.NewPostgresRepo(pool),
		CurrencyRates: currencystorage.NewPostgresRateRepo(pool),
		Sequence:      sequencestorage.NewPostgresRepo(pool),
		Attachment:    attachmentstorage.NewPostgresRepo(pool),
		BankStatement: bankstatementstorage.NewPostgresRepo(pool),
		Project:       projectstorage.NewPostgresRepo(pool),
		Permission:    groupstorage.NewPostgresRepo(pool),
		MRP:           mrpstorage.NewPostgresRepo(pool),
		Requisition:   purchasestorage.NewRequisitionPostgresRepo(pool),
		Activity:      activityRepo,
		ActivityType:  activityRepo,
		ActivityMsg:   activityRepo,
		ActivityNotif: activityRepo,
		EmailQueue:    activityRepo,
	}
}

func NewFromMemory() *CashflowRepositories {
	activityRepo := activitystorage.NewMemoryRepo()
	return &CashflowRepositories{
		Partner:       partnerstorage.NewMemoryRepo(),
		Product:       productstorage.NewMemoryRepo(),
		Accounting:    accountingstorage.NewMemoryRepo(),
		Analytic:      analyticstorage.NewMemoryRepo(),
		Sale:          salestorage.NewMemoryRepo(),
		Purchase:      purchasestorage.NewMemoryRepo(),
		Stock:         stockstorage.NewMemoryRepo(),
		CRM:           crmstorage.NewMemoryRepo(),
		Payment:       paymentstorage.NewMemoryRepo(),
		HR:            hrstorage.NewMemoryRepo(),
		Company:       companystorage.NewMemoryRepo(),
		User:          userstorage.NewMemoryRepo(),
		Currency:      currencystorage.NewMemoryRepo(),
		CurrencyRates: currencystorage.NewMemoryRateRepo(),
		Sequence:      sequencestorage.NewMemoryRepo(),
		Attachment:    attachmentstorage.NewMemoryRepo(),
		BankStatement: bankstatementstorage.NewMemoryRepo(),
		Project:       projectstorage.NewMemoryRepo(),
		Permission:    groupstorage.NewMemoryRepo(),
		MRP:           mrpstorage.NewMemoryRepo(),
		Requisition:   purchasestorage.NewMemoryRequisitionRepo(),
		Activity:      activityRepo,
		ActivityType:  activityRepo,
		ActivityMsg:   activityRepo,
		ActivityNotif: activityRepo,
		EmailQueue:    activityRepo,
	}
}
