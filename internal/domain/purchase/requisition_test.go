package purchase

import "testing"

func validRequisition() PurchaseRequisition {
	vendorID := int64(10)
	return PurchaseRequisition{
		Type:     RequisitionBlanketOrder,
		State:    RequisitionDraft,
		VendorID: &vendorID,
		Lines: []PurchaseRequisitionLine{{
			ProductID: 1,
			ProductQty: 2,
			PriceUnit: 5,
		}},
	}
}

func TestPurchaseRequisitionValidateRejectsUnknownType(t *testing.T) {
	requisition := validRequisition()
	requisition.Type = RequisitionType("unknown")

	if err := requisition.Validate(); err == nil {
		t.Fatal("expected unknown requisition type to be rejected")
	}
}

func TestPurchaseRequisitionValidateRejectsUnknownState(t *testing.T) {
	requisition := validRequisition()
	requisition.State = RequisitionState("unknown")

	if err := requisition.Validate(); err == nil {
		t.Fatal("expected unknown requisition state to be rejected")
	}
}

func TestPurchaseRequisitionConfirmRequiresBlanketVendorAndPrice(t *testing.T) {
	withoutVendor := validRequisition()
	withoutVendor.VendorID = nil
	if err := withoutVendor.Confirm(); err == nil {
		t.Fatal("expected blanket order without vendor to be rejected")
	}

	withoutPrice := validRequisition()
	withoutPrice.Lines[0].PriceUnit = 0
	if err := withoutPrice.Confirm(); err == nil {
		t.Fatal("expected blanket order without a positive price to be rejected")
	}
}

func TestPurchaseRequisitionCancelRejectsFinalState(t *testing.T) {
	requisition := validRequisition()
	requisition.State = RequisitionDone

	if err := requisition.Cancel(); err == nil {
		t.Fatal("expected done requisition to reject cancellation")
	}
}

func TestPurchaseRequisitionCanChangeDefinitionOnlyInDraft(t *testing.T) {
	requisition := validRequisition()
	if err := requisition.CanChangeDefinition(); err != nil {
		t.Fatalf("draft requisition should allow definition changes: %v", err)
	}

	requisition.State = RequisitionConfirmed
	if err := requisition.CanChangeDefinition(); err == nil {
		t.Fatal("expected confirmed requisition to reject definition changes")
	}
}