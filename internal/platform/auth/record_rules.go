package auth

import (
	"context"
	"fmt"
	"reflect"
)

// Record is the small, typed input exposed to a record-rule evaluator.
type Record map[string]any

// RuleField limits fields that may be referenced by a record rule.
type RuleField string

const (
	RuleFieldID        RuleField = "id"
	RuleFieldCompanyID RuleField = "company_id"
	RuleFieldCreatedBy RuleField = "created_by"
	RuleFieldOwnerID   RuleField = "owner_id"
)

// ValueRef resolves a literal or a small set of request-scoped values.
type ValueRef struct {
	Literal any
	Ref     string
}

// Predicate is a safe comparison against an allow-listed field.
type Predicate struct {
	Field    RuleField
	Operator string
	Value    ValueRef
}

// RuleNode is a typed record-rule expression. Implementations must not execute code.
type RuleNode interface {
	evaluate(context.Context, Subject, Record) (bool, error)
}

type And struct{ Nodes []RuleNode }
type Or struct{ Nodes []RuleNode }
type Not struct{ Node RuleNode }
type Condition struct{ Predicate Predicate }

func (n And) evaluate(ctx context.Context, subject Subject, record Record) (bool, error) {
	for _, child := range n.Nodes {
		matched, err := child.evaluate(ctx, subject, record)
		if err != nil || !matched {
			return matched, err
		}
	}
	return true, nil
}

func (n Or) evaluate(ctx context.Context, subject Subject, record Record) (bool, error) {
	for _, child := range n.Nodes {
		matched, err := child.evaluate(ctx, subject, record)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (n Not) evaluate(ctx context.Context, subject Subject, record Record) (bool, error) {
	if n.Node == nil {
		return false, fmt.Errorf("record rule NOT node cannot be nil")
	}
	matched, err := n.Node.evaluate(ctx, subject, record)
	return !matched, err
}

func (n Condition) evaluate(ctx context.Context, subject Subject, record Record) (bool, error) {
	value, ok := record[string(n.Predicate.Field)]
	if !ok {
		return false, nil
	}
	expected := resolveValue(n.Predicate.Value, subject)
	switch n.Predicate.Operator {
	case "eq":
		return reflect.DeepEqual(value, expected), nil
	case "ne":
		return !reflect.DeepEqual(value, expected), nil
	case "in":
		return contains(expected, value), nil
	default:
		return false, fmt.Errorf("unsupported record rule operator %q", n.Predicate.Operator)
	}
}

// EvaluateRecordRules applies global rules with AND and group rules with OR.
func EvaluateRecordRules(ctx context.Context, subject Subject, record Record, global []RuleNode, group []RuleNode) (bool, error) {
	if subject.IsSuperuser {
		return true, nil
	}
	for _, rule := range global {
		matched, err := rule.evaluate(ctx, subject, record)
		if err != nil || !matched {
			return matched, err
		}
	}
	if len(group) == 0 {
		return true, nil
	}
	return Or{Nodes: group}.evaluate(ctx, subject, record)
}

func resolveValue(value ValueRef, subject Subject) any {
	switch value.Ref {
	case "current_user":
		return subject.UserID
	case "current_company":
		return subject.CompanyID
	default:
		return value.Literal
	}
}

func contains(values any, value any) bool {
	items := reflect.ValueOf(values)
	if !items.IsValid() || (items.Kind() != reflect.Slice && items.Kind() != reflect.Array) {
		return false
	}
	for i := 0; i < items.Len(); i++ {
		if reflect.DeepEqual(items.Index(i).Interface(), value) {
			return true
		}
	}
	return false
}
