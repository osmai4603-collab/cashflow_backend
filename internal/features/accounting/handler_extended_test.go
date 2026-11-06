package accounting

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestExtendedAccountingHandlers(t *testing.T) {
	repo := newMockAccountingRepository()
	server := newAccountingTestServer(repo)
	defer server.Close()
	client := server.Client()

	// 1. POST /api/v1/accounting/journals
	createJournalRes := postJSON(t, client, server.URL+"/api/v1/accounting/journals", CreateJournalRequest{
		Code:           "BANK1",
		Name:           "Main Bank",
		Type:           "bank",
		SequencePrefix: "BNK/2026/",
	})
	if createJournalRes.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createJournalRes.Body)
		t.Fatalf("create journal status = %d, want %d; body: %s", createJournalRes.StatusCode, http.StatusCreated, body)
	}
	var createdJournal JournalResponse
	_ = json.NewDecoder(createJournalRes.Body).Decode(&createdJournal)
	createJournalRes.Body.Close()

	if createdJournal.Code != "BANK1" {
		t.Errorf("expected code BANK1, got %s", createdJournal.Code)
	}

	// 2. PUT /api/v1/accounting/journals/{id}
	putBody, _ := json.Marshal(UpdateJournalRequest{
		Name:           "Main Bank Updated",
		SequencePrefix: "BNK/2026/V2/",
	})
	putReq, _ := http.NewRequest(http.MethodPut, server.URL+"/api/v1/accounting/journals/"+createdJournal.ID.String(), bytes.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putRes, err := client.Do(putReq)
	if err != nil {
		t.Fatalf("put journal: %v", err)
	}
	if putRes.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putRes.Body)
		t.Fatalf("update journal status = %d, want %d; body: %s", putRes.StatusCode, http.StatusOK, body)
	}
	putRes.Body.Close()

	// 3. GET /api/v1/accounting/governance/verify-integrity
	verifyRes, err := client.Get(server.URL + "/api/v1/accounting/governance/verify-integrity")
	if err != nil {
		t.Fatalf("verify integrity: %v", err)
	}
	if verifyRes.StatusCode != http.StatusOK {
		t.Fatalf("verify integrity status = %d, want %d", verifyRes.StatusCode, http.StatusOK)
	}
	var integrityRes AuditIntegrityResponse
	_ = json.NewDecoder(verifyRes.Body).Decode(&integrityRes)
	verifyRes.Body.Close()
	if !integrityRes.IsValid {
		t.Error("expected integrity to be valid")
	}

	// 4. GET /api/v1/accounting/governance/audit-logs
	logsRes, err := client.Get(server.URL + "/api/v1/accounting/governance/audit-logs?limit=20&offset=0")
	if err != nil {
		t.Fatalf("get audit logs: %v", err)
	}
	if logsRes.StatusCode != http.StatusOK {
		t.Fatalf("audit logs status = %d, want %d", logsRes.StatusCode, http.StatusOK)
	}
	logsRes.Body.Close()

	// 5. GET & PUT /api/v1/accounting/configurations
	retainedAcc := &Account{ID: uuid.New(), Code: "300000", Name: "Retained", Type: AccountTypeEquity, IsActive: true}
	fxGainAcc := &Account{ID: uuid.New(), Code: "450000", Name: "Gain", Type: AccountTypeIncome, IsActive: true}
	fxLossAcc := &Account{ID: uuid.New(), Code: "550000", Name: "Loss", Type: AccountTypeExpense, IsActive: true}
	repo.accountsByID[retainedAcc.ID] = retainedAcc
	repo.accountsByID[fxGainAcc.ID] = fxGainAcc
	repo.accountsByID[fxLossAcc.ID] = fxLossAcc

	configBody, _ := json.Marshal(UpdateAccountingConfigurationRequest{
		RetainedEarningsAccountID: retainedAcc.ID,
		FXGainAccountID:           fxGainAcc.ID,
		FXLossAccountID:           fxLossAcc.ID,
		FXReconciliationJournalID: createdJournal.ID,
		FunctionalCurrency:        "SAR",
	})
	configReq, _ := http.NewRequest(http.MethodPut, server.URL+"/api/v1/accounting/configurations", bytes.NewReader(configBody))
	configReq.Header.Set("Content-Type", "application/json")
	configPutRes, err := client.Do(configReq)
	if err != nil {
		t.Fatalf("update config: %v", err)
	}
	if configPutRes.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(configPutRes.Body)
		t.Fatalf("update config status = %d, want %d; body: %s", configPutRes.StatusCode, http.StatusOK, body)
	}
	configPutRes.Body.Close()

	configGetRes, err := client.Get(server.URL + "/api/v1/accounting/configurations")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if configGetRes.StatusCode != http.StatusOK {
		t.Fatalf("get config status = %d, want %d", configGetRes.StatusCode, http.StatusOK)
	}
	configGetRes.Body.Close()

	// 6. Subledger Mappings & Events
	debitAcc := &Account{ID: uuid.New(), Code: "500000", Name: "Purchases", Type: AccountTypeExpense, IsActive: true}
	creditAcc := &Account{ID: uuid.New(), Code: "201000", Name: "AP", Type: AccountTypeLiability, IsActive: true}
	repo.accountsByID[debitAcc.ID] = debitAcc
	repo.accountsByID[creditAcc.ID] = creditAcc

	saveMapRes := postJSON(t, client, server.URL+"/api/v1/accounting/subledger/mappings", SaveSubledgerEventMappingRequest{
		EventType:       "vendor_bill_received",
		JournalID:       createdJournal.ID,
		DebitAccountID:  debitAcc.ID,
		CreditAccountID: creditAcc.ID,
		Description:     "Auto bill mapping",
		IsActive:        true,
	})
	if saveMapRes.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(saveMapRes.Body)
		t.Fatalf("save mapping status = %d, want %d; body: %s", saveMapRes.StatusCode, http.StatusOK, body)
	}
	saveMapRes.Body.Close()

	listMapRes, err := client.Get(server.URL + "/api/v1/accounting/subledger/mappings")
	if err != nil {
		t.Fatalf("list mappings: %v", err)
	}
	if listMapRes.StatusCode != http.StatusOK {
		t.Fatalf("list mappings status = %d, want %d", listMapRes.StatusCode, http.StatusOK)
	}
	listMapRes.Body.Close()

	eventsRes, err := client.Get(server.URL + "/api/v1/accounting/subledger/events?limit=10")
	if err != nil {
		t.Fatalf("get subledger events: %v", err)
	}
	if eventsRes.StatusCode != http.StatusOK {
		t.Fatalf("subledger events status = %d, want %d", eventsRes.StatusCode, http.StatusOK)
	}
	eventsRes.Body.Close()
}

