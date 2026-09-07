package auth

import (
	"context"
	"testing"
)

func TestEvaluateRecordRulesCombinesGlobalAndGroupRules(t *testing.T) {
	ctx := context.Background()
	subject := Subject{UserID: 7, CompanyID: 3}
	record := Record{"company_id": int64(3), "owner_id": int64(7)}
	global := []RuleNode{Condition{Predicate: Predicate{
		Field: RuleFieldCompanyID, Operator: "eq", Value: ValueRef{Ref: "current_company"},
	}}}
	groups := []RuleNode{
		Condition{Predicate: Predicate{Field: RuleFieldOwnerID, Operator: "eq", Value: ValueRef{Ref: "current_user"}}},
		Condition{Predicate: Predicate{Field: RuleFieldID, Operator: "eq", Value: ValueRef{Literal: int64(99)}}},
	}

	allowed, err := EvaluateRecordRules(ctx, subject, record, global, groups)
	if err != nil || !allowed {
		t.Fatalf("expected global AND matching group rule, allowed=%v err=%v", allowed, err)
	}

	record["company_id"] = int64(8)
	allowed, err = EvaluateRecordRules(ctx, subject, record, global, groups)
	if err != nil || allowed {
		t.Fatalf("expected global rule to deny, allowed=%v err=%v", allowed, err)
	}
}

func TestEvaluateRecordRulesSuperuserBypassesRules(t *testing.T) {
	allowed, err := EvaluateRecordRules(context.Background(), Subject{UserID: 1, IsSuperuser: true}, Record{}, []RuleNode{
		Condition{Predicate: Predicate{Field: RuleFieldCompanyID, Operator: "eq", Value: ValueRef{Literal: int64(999)}}},
	}, nil)
	if err != nil || !allowed {
		t.Fatalf("expected superuser bypass, allowed=%v err=%v", allowed, err)
	}
}
