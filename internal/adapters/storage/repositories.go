package storage

import (
	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	activitystorage "cashflow_backend/internal/adapters/storage/activity"
	analyticstorage "cashflow_backend/internal/adapters/storage/analytic"
	attachmentstorage "cashflow_backend/internal/adapters/storage/attachment"
	bankstatementstorage "cashflow_backend/internal/adapters/storage/bankstatement"
	calendarstorage "cashflow_backend/internal/adapters/storage/calendar"
	companystorage "cashflow_backend/internal/adapters/storage/company"
	crmstorage "cashflow_backend/internal/adapters/storage/crm"
	currencystorage "cashflow_backend/internal/adapters/storage/currency"
	deliverystorage "cashflow_backend/internal/adapters/storage/delivery"
	ecommercestorage "cashflow_backend/internal/adapters/storage/ecommerce"
	expensestorage "cashflow_backend/internal/adapters/storage/expense"
	fleetstorage "cashflow_backend/internal/adapters/storage/fleet"
	groupstorage "cashflow_backend/internal/adapters/storage/group"
	helpdeskstorage "cashflow_backend/internal/adapters/storage/helpdesk"
	hrstorage "cashflow_backend/internal/adapters/storage/hr"
	livechatstorage "cashflow_backend/internal/adapters/storage/livechat"
	loyaltystorage "cashflow_backend/internal/adapters/storage/loyalty"
	maintenancestorage "cashflow_backend/internal/adapters/storage/maintenance"
	marketingstorage "cashflow_backend/internal/adapters/storage/marketing"
	mrpstorage "cashflow_backend/internal/adapters/storage/mrp"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	paymentstorage "cashflow_backend/internal/adapters/storage/payment"
	planningstorage "cashflow_backend/internal/adapters/storage/planning"
	portalstorage "cashflow_backend/internal/adapters/storage/portal"
	posstorage "cashflow_backend/internal/adapters/storage/pos"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	projectstorage "cashflow_backend/internal/adapters/storage/project"
	purchasestorage "cashflow_backend/internal/adapters/storage/purchase"
	qualitystorage "cashflow_backend/internal/adapters/storage/quality"
	recruitmentstorage "cashflow_backend/internal/adapters/storage/recruitment"
	repairstorage "cashflow_backend/internal/adapters/storage/repair"
	reportstorage "cashflow_backend/internal/adapters/storage/report"
	resourcestorage "cashflow_backend/internal/adapters/storage/resource"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	sequencestorage "cashflow_backend/internal/adapters/storage/sequence"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	subscriptionstorage "cashflow_backend/internal/adapters/storage/subscription"
	surveystorage "cashflow_backend/internal/adapters/storage/survey"
	timesheetstorage "cashflow_backend/internal/adapters/storage/timesheet"
	userstorage "cashflow_backend/internal/adapters/storage/user"
	websitestorage "cashflow_backend/internal/adapters/storage/website"
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/domain/analytic"
	"cashflow_backend/internal/domain/attachment"
	"cashflow_backend/internal/domain/bankstatement"
	"cashflow_backend/internal/domain/calendar"
	"cashflow_backend/internal/domain/company"
	"cashflow_backend/internal/domain/crm"
	"cashflow_backend/internal/domain/currency"
	"cashflow_backend/internal/domain/delivery"
	"cashflow_backend/internal/domain/ecommerce"
	"cashflow_backend/internal/domain/expense"
	"cashflow_backend/internal/domain/fleet"
	"cashflow_backend/internal/domain/group"
	"cashflow_backend/internal/domain/helpdesk"
	"cashflow_backend/internal/domain/hr"
	"cashflow_backend/internal/domain/livechat"
	"cashflow_backend/internal/domain/loyalty"
	"cashflow_backend/internal/domain/maintenance"
	"cashflow_backend/internal/domain/marketing"
	"cashflow_backend/internal/domain/mrp"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/payment"
	"cashflow_backend/internal/domain/planning"
	"cashflow_backend/internal/domain/portal"
	"cashflow_backend/internal/domain/pos"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/project"
	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/domain/quality"
	"cashflow_backend/internal/domain/recruitment"
	"cashflow_backend/internal/domain/repair"
	"cashflow_backend/internal/domain/report"
	"cashflow_backend/internal/domain/resource"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/domain/sequence"
	"cashflow_backend/internal/domain/stock"
	"cashflow_backend/internal/domain/subscription"
	"cashflow_backend/internal/domain/survey"
	"cashflow_backend/internal/domain/timesheet"
	"cashflow_backend/internal/domain/user"
	"cashflow_backend/internal/domain/website"

	purchaseusecase "cashflow_backend/internal/usecase/purchase"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CashflowRepositories struct {
	Partner          partner.Repository
	Product          product.Repository
	Accounting       accounting.Repository
	Analytic         analytic.Repository
	Sale             sale.Repository
	Purchase         purchase.Repository
	Stock            stock.Repository
	CRM              crm.Repository
	Expense          expense.Repository
	Payment          payment.Repository
	Pos              pos.Repository
	Recruitment      recruitment.Repository
	Timesheet        timesheet.Repository
	Resource         resource.Repository
	HR               hr.Repository
	Company          company.Repository
	User             user.Repository
	Currency         currency.Repository
	CurrencyRates    currency.RateRepository
	Sequence         sequence.Repository
	Attachment       attachment.Repository
	BankStatement    bankstatement.Repository
	Calendar         calendar.Repository
	Project          project.Repository
	Permission       group.PermissionRepository
	Report           report.Repository
	ReportData       report.DataRepository
	MRP              mrp.Repository
	Requisition      purchaseusecase.RequisitionRepository
	SupplierInfo     purchaseusecase.SupplierInfoRepository
	Maintenance      maintenance.Repository
	Fleet            fleet.Repository
	Loyalty          loyalty.Repository
	Delivery         delivery.Repository
	Activity         activity.ActivityRepository
	ActivityType     activity.ActivityTypeRepository
	ActivityMsg      activity.MessageRepository
	ActivityNotif    activity.NotificationRepository
	ActivityFollower activity.FollowerRepository
	ActivitySubtype  activity.SubtypeRepository
	EmailQueue       activity.EmailQueueRepository
	Website          website.Repository
	Ecommerce        ecommerce.Repository
	Portal           portal.Repository
	MarketingCampaign    marketing.CampaignRepository
	MarketingMailingList marketing.MailingListRepository
	MarketingContact     marketing.ContactRepository
	MarketingMassMailing marketing.MassMailingRepository
	MarketingTracking    marketing.TrackingRepository
	MarketingAutomation   marketing.AutomationRepository
	LiveChat             livechat.Repository
	Helpdesk             helpdesk.Repository
	Subscription         subscription.Repository
	Quality              quality.Repository
	Survey               survey.Repository
	Repair               repair.Repository
	Planning             planning.Repository
}

func NewFromPostgres(pool *pgxpool.Pool) *CashflowRepositories {
	activityRepo := activitystorage.NewPostgresRepo(pool)
	requisitionRepo := purchasestorage.NewRequisitionPostgresRepo(pool)
	reportRepo := reportstorage.NewPostgresRepo(pool)
	marketingRepo := marketingstorage.NewPostgresRepo(pool)
	return &CashflowRepositories{
		Partner:          partnerstorage.NewPostgresRepo(pool),
		Product:          productstorage.NewPostgresRepo(pool),
		Accounting:       accountingstorage.NewPostgresRepo(pool),
		Analytic:         analyticstorage.NewPostgresRepo(pool),
		Sale:             salestorage.NewPostgresRepo(pool),
		Purchase:         purchasestorage.NewPostgresRepo(pool),
		Stock:            stockstorage.NewPostgresRepo(pool),
		CRM:              crmstorage.NewPostgresRepo(pool),
		Expense:          expensestorage.NewPostgresRepo(pool),
		Payment:          paymentstorage.NewPostgresRepo(pool),
		Pos:              posstorage.NewPostgresRepo(pool),
		Recruitment:      recruitmentstorage.NewPostgresRepo(pool),
		Timesheet:        timesheetstorage.NewPostgresRepo(pool),
		Resource:         resourcestorage.NewPostgresRepo(pool),
		HR:               hrstorage.NewPostgresRepo(pool),
		Company:          companystorage.NewPostgresRepo(pool),
		User:             userstorage.NewPostgresRepo(pool),
		Currency:         currencystorage.NewPostgresRepo(pool),
		CurrencyRates:    currencystorage.NewPostgresRateRepo(pool),
		Sequence:         sequencestorage.NewPostgresRepo(pool),
		Attachment:       attachmentstorage.NewPostgresRepo(pool),
		BankStatement:    bankstatementstorage.NewPostgresRepo(pool),
		Calendar:         calendarstorage.NewPostgresRepo(pool),
		Project:          projectstorage.NewPostgresRepo(pool),
		Permission:       groupstorage.NewPostgresRepo(pool),
		Report:           reportRepo,
		ReportData:       reportRepo,
		MRP:              mrpstorage.NewPostgresRepo(pool),
		Requisition:      requisitionRepo,
		SupplierInfo:     requisitionRepo,
		Maintenance:      maintenancestorage.NewPostgresRepo(pool),
		Fleet:            fleetstorage.NewPostgresRepo(pool),
		Loyalty:          loyaltystorage.NewPostgresRepo(pool),
		Delivery:         deliverystorage.NewPostgresRepository(pool),
		Activity:         activityRepo,
		ActivityType:     activityRepo,
		ActivityMsg:      activityRepo,
		ActivityNotif:    activityRepo,
		ActivityFollower: activityRepo,
		ActivitySubtype:  activityRepo,
		EmailQueue:       activityRepo,
		Website:          websitestorage.NewPostgresRepo(pool),
		Ecommerce:        ecommercestorage.NewPostgresRepo(pool),
		Portal:           portalstorage.NewPostgresRepo(pool),
		MarketingCampaign:    marketingRepo,
		MarketingMailingList: marketingRepo,
		MarketingContact:     marketingRepo,
		MarketingMassMailing: marketingRepo,
		MarketingTracking:    marketingRepo,
		MarketingAutomation:   marketingRepo,
		LiveChat:             livechatstorage.NewPostgresRepo(pool),
		Helpdesk:             helpdeskstorage.NewPostgresRepo(pool),
		Subscription:         subscriptionstorage.NewPostgresRepo(pool),
		Quality:              qualitystorage.NewPostgresRepo(pool),
		Survey:               surveystorage.NewPostgresRepo(pool),
		Repair:               repairstorage.NewPostgresRepo(pool),
		Planning:             planningstorage.NewPostgresRepo(pool),
	}
}

func NewFromMemory() *CashflowRepositories {
	activityRepo := activitystorage.NewMemoryRepo()
	requisitionRepo := purchasestorage.NewMemoryRequisitionRepo()
	reportRepo := reportstorage.NewMemoryRepo()
	marketingRepo := marketingstorage.NewMemoryRepo()
	return &CashflowRepositories{
		Partner:          partnerstorage.NewMemoryRepo(),
		Product:          productstorage.NewMemoryRepo(),
		Accounting:       accountingstorage.NewMemoryRepo(),
		Analytic:         analyticstorage.NewMemoryRepo(),
		Sale:             salestorage.NewMemoryRepo(),
		Purchase:         purchasestorage.NewMemoryRepo(),
		Stock:            stockstorage.NewMemoryRepo(),
		CRM:              crmstorage.NewMemoryRepo(),
		Expense:          expensestorage.NewMemoryRepo(),
		Payment:          paymentstorage.NewMemoryRepo(),
		Pos:              posstorage.NewMemoryRepo(),
		Recruitment:      recruitmentstorage.NewMemoryRepo(),
		Timesheet:        timesheetstorage.NewMemoryRepo(),
		Resource:         resourcestorage.NewMemoryRepo(),
		HR:               hrstorage.NewMemoryRepo(),
		Company:          companystorage.NewMemoryRepo(),
		User:             userstorage.NewMemoryRepo(),
		Currency:         currencystorage.NewMemoryRepo(),
		CurrencyRates:    currencystorage.NewMemoryRateRepo(),
		Sequence:         sequencestorage.NewMemoryRepo(),
		Attachment:       attachmentstorage.NewMemoryRepo(),
		BankStatement:    bankstatementstorage.NewMemoryRepo(),
		Calendar:         calendarstorage.NewMemoryRepo(),
		Project:          projectstorage.NewMemoryRepo(),
		Permission:       groupstorage.NewMemoryRepo(),
		Report:           reportRepo,
		ReportData:       reportRepo,
		MRP:              mrpstorage.NewMemoryRepo(),
		Requisition:      requisitionRepo,
		SupplierInfo:     requisitionRepo,
		Maintenance:      maintenancestorage.NewMemoryRepo(),
		Fleet:            fleetstorage.NewMemoryRepo(),
		Loyalty:          loyaltystorage.NewMemoryRepo(),
		Delivery:         deliverystorage.NewMemoryRepository(),
		Activity:         activityRepo,
		ActivityType:     activityRepo,
		ActivityMsg:      activityRepo,
		ActivityNotif:    activityRepo,
		ActivityFollower: activityRepo,
		ActivitySubtype:  activityRepo,
		EmailQueue:       activityRepo,
		Website:          websitestorage.NewMemoryRepo(),
		Ecommerce:        ecommercestorage.NewMemoryRepo(),
		Portal:           portalstorage.NewMemoryRepo(),
		MarketingCampaign:    marketingRepo,
		MarketingMailingList: marketingRepo,
		MarketingContact:     marketingRepo,
		MarketingMassMailing: marketingRepo,
		MarketingTracking:    marketingRepo,
		MarketingAutomation:   marketingRepo,
		LiveChat:             livechatstorage.NewMemoryRepo(),
		Helpdesk:             helpdeskstorage.NewMemoryRepo(),
		Subscription:         subscriptionstorage.NewMemoryRepo(),
		Quality:              qualitystorage.NewMemoryRepo(),
		Survey:               surveystorage.NewMemoryRepo(),
		Repair:               repairstorage.NewMemoryRepo(),
		Planning:             planningstorage.NewMemoryRepo(),
	}
}
