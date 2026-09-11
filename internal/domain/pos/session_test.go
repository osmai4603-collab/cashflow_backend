package pos

import "testing"

func TestPosSessionLifecycleAndCashDifference(t *testing.T) {
	session := &PosSession{ConfigID: 1, UserID: 2, CompanyID: 3}
	if err := session.Validate(); err != nil {
		t.Fatalf("validate session: %v", err)
	}
	if err := session.Open(100); err != nil {
		t.Fatalf("open session: %v", err)
	}
	if err := session.BeginClosing(125.50); err != nil {
		t.Fatalf("begin closing: %v", err)
	}
	if err := session.Close(120.25); err != nil {
		t.Fatalf("close session: %v", err)
	}
	if session.State != SessionStateClosed || session.CashRegisterDifference != -5.25 {
		t.Fatalf("unexpected closing result: state=%s difference=%v", session.State, session.CashRegisterDifference)
	}
}

func TestPosSessionCannotCloseBeforeClosingControl(t *testing.T) {
	session := &PosSession{ConfigID: 1, UserID: 2, CompanyID: 3}
	if err := session.Open(0); err != nil {
		t.Fatalf("open session: %v", err)
	}
	if err := session.Close(10); err == nil {
		t.Fatal("expected closing an opened session to fail")
	}
}
