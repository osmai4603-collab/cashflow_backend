package filter

import (
	"fmt"
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// Supported Operators
const (
	OpEqual              = "eq"
	OpNotEqual           = "ne"
	OpGreaterThan        = "gt"
	OpGreaterThanOrEqual = "gte"
	OpLessThan           = "lt"
	OpLessThanOrEqual    = "lte"
	OpLike               = "like"
	OpILike              = "ilike"
	OpIn                 = "in"
	OpIsNull             = "is_null"
	OpIsNotNull          = "is_not_null"
)

// Criterion represents a single filtering condition inspired by Odoo domains.
type Criterion struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// Filter is a collection of Criteria combined with AND logic.
type Filter struct {
	Criteria []Criterion `json:"criteria"`
}

// NewFilter creates a filter with the given criteria.
func NewFilter(criteria ...Criterion) *Filter {
	return &Filter{Criteria: criteria}
}

// Add appends a new criterion to the filter.
func (f *Filter) Add(field, op string, val any) *Filter {
	f.Criteria = append(f.Criteria, Criterion{
		Field:    field,
		Operator: op,
		Value:    val,
	})
	return f
}

// BuildWhereClause constructs a parameterized SQL WHERE clause using allowedFields whitelist.
// Returns:
// - whereClause: e.g. "WHERE column1 = $1 AND column2 ILIKE $2"
// - args: slice of values corresponding to $1, $2, etc.
// - nextIndex: next available placeholder index
func (f *Filter) BuildWhereClause(allowedFields map[string]string, startingIndex int) (string, []any, int, error) {
	if f == nil || len(f.Criteria) == 0 {
		return "", nil, startingIndex, nil
	}

	var clauses []string
	var args []any
	idx := startingIndex
	if idx <= 0 {
		idx = 1
	}

	for _, c := range f.Criteria {
		colName, allowed := allowedFields[c.Field]
		if !allowed {
			return "", nil, idx, platformerrors.BadRequest(fmt.Sprintf("filtering on field %q is not permitted", c.Field))
		}

		op := strings.ToLower(strings.TrimSpace(c.Operator))
		switch op {
		case OpEqual, "=":
			clauses = append(clauses, fmt.Sprintf("%s = $%d", colName, idx))
			args = append(args, c.Value)
			idx++

		case OpNotEqual, "!=", "<>":
			clauses = append(clauses, fmt.Sprintf("%s != $%d", colName, idx))
			args = append(args, c.Value)
			idx++

		case OpGreaterThan, ">":
			clauses = append(clauses, fmt.Sprintf("%s > $%d", colName, idx))
			args = append(args, c.Value)
			idx++

		case OpGreaterThanOrEqual, ">=":
			clauses = append(clauses, fmt.Sprintf("%s >= $%d", colName, idx))
			args = append(args, c.Value)
			idx++

		case OpLessThan, "<":
			clauses = append(clauses, fmt.Sprintf("%s < $%d", colName, idx))
			args = append(args, c.Value)
			idx++

		case OpLessThanOrEqual, "<=":
			clauses = append(clauses, fmt.Sprintf("%s <= $%d", colName, idx))
			args = append(args, c.Value)
			idx++

		case OpLike:
			clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", colName, idx))
			args = append(args, fmt.Sprintf("%%%v%%", c.Value))
			idx++

		case OpILike:
			clauses = append(clauses, fmt.Sprintf("%s ILIKE $%d", colName, idx))
			args = append(args, fmt.Sprintf("%%%v%%", c.Value))
			idx++

		case OpIsNull:
			clauses = append(clauses, fmt.Sprintf("%s IS NULL", colName))

		case OpIsNotNull:
			clauses = append(clauses, fmt.Sprintf("%s IS NOT NULL", colName))

		case OpIn:
			// Expects slice or array
			sliceVal, ok := c.Value.([]any)
			if !ok {
				// Also handle string slice
				if strSlice, okStr := c.Value.([]string); okStr {
					sliceVal = make([]any, len(strSlice))
					for i, s := range strSlice {
						sliceVal[i] = s
					}
				} else if intSlice, okInt := c.Value.([]int); okInt {
					sliceVal = make([]any, len(intSlice))
					for i, v := range intSlice {
						sliceVal[i] = v
					}
				} else if int64Slice, okInt64 := c.Value.([]int64); okInt64 {
					sliceVal = make([]any, len(int64Slice))
					for i, v := range int64Slice {
						sliceVal[i] = v
					}
				}
			}

			if len(sliceVal) == 0 {
				clauses = append(clauses, "FALSE")
				continue
			}

			var placeholders []string
			for _, item := range sliceVal {
				placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
				args = append(args, item)
				idx++
			}
			clauses = append(clauses, fmt.Sprintf("%s IN (%s)", colName, strings.Join(placeholders, ", ")))

		default:
			return "", nil, idx, platformerrors.BadRequest(fmt.Sprintf("unsupported operator %q", c.Operator))
		}
	}

	if len(clauses) == 0 {
		return "", nil, idx, nil
	}

	whereClause := "WHERE " + strings.Join(clauses, " AND ")
	return whereClause, args, idx, nil
}
