package marketingusecase

import (
	"context"
	"testing"

	marketingstorage "cashflow_backend/internal/adapters/storage/marketing"
	"cashflow_backend/internal/domain/marketing"
)

func TestMarketingUseCase_CampaignListsContactsAndBlacklist(t *testing.T) {
	repo := marketingstorage.NewMemoryRepo()
	uc := New(repo, repo, repo, repo, repo, repo)
	ctx := context.Background()

	campaign, err := uc.CreateCampaign(ctx, &marketing.Campaign{Name: "Spring Sale"})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if campaign.ID == 0 || campaign.CompanyID != defaultMarketingCompanyID || campaign.UTMCampaign != "spring-sale" {
		t.Fatalf("campaign defaults were not applied: %#v", campaign)
	}

	campaigns, err := uc.ListCampaigns(ctx)
	if err != nil {
		t.Fatalf("list campaigns: %v", err)
	}
	if len(campaigns) != 1 || campaigns[0].ID != campaign.ID {
		t.Fatalf("unexpected campaigns: %#v", campaigns)
	}

	list, err := uc.CreateMailingList(ctx, &marketing.MailingList{Name: "Customers"})
	if err != nil {
		t.Fatalf("create mailing list: %v", err)
	}
	csvData := []byte("email,name,mobile\nuser@example.com,User,+966500000000\nuser@example.com,User,+966500000000\n")
	if err := uc.ImportContacts(ctx, list.ID, csvData); err != nil {
		t.Fatalf("import contacts: %v", err)
	}
	contacts, err := repo.ListContacts(ctx, defaultMarketingCompanyID)
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(contacts) != 1 || contacts[0].Email != "user@example.com" {
		t.Fatalf("duplicate contacts were not coalesced: %#v", contacts)
	}

	if err := uc.AddToBlacklist(ctx, "Blocked@example.com"); err != nil {
		t.Fatalf("add blacklist entry: %v", err)
	}
	blacklist, err := uc.ListBlacklist(ctx)
	if err != nil {
		t.Fatalf("list blacklist: %v", err)
	}
	if len(blacklist) != 1 || blacklist[0].Value != "blocked@example.com" {
		t.Fatalf("unexpected blacklist: %#v", blacklist)
	}
}
