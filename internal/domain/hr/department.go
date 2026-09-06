package hr

import (
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// Department represents an organizational unit or division (hr.department in Odoo).
type Department struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	CompleteName string       `json:"complete_name"` // Hierarchical path e.g. "Management / Sales"
	ParentID     *int64       `json:"parent_id,omitempty"`
	ManagerID    *int64       `json:"manager_id,omitempty"`
	CompanyID    *int64       `json:"company_id,omitempty"`
	Color        int          `json:"color"`
	Active       bool         `json:"active"`
	Audit        audit.Fields `json:"audit"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	CreatedBy    *int64       `json:"created_by,omitempty"`
	UpdatedBy    *int64       `json:"updated_by,omitempty"`
}

// DepartmentNode represents an organizational department in a hierarchical tree view.
type DepartmentNode struct {
	Department
	Children      []DepartmentNode `json:"children,omitempty"`
	EmployeeCount int              `json:"employee_count"`
}

// Validate checks business invariants on the Department entity.
func (d *Department) Validate() error {
	d.Name = strings.TrimSpace(d.Name)
	if d.Name == "" {
		return platformerrors.Validation("department name is required", map[string]string{
			"name": "cannot be empty",
		})
	}

	if d.ParentID != nil && d.ID > 0 && *d.ParentID == d.ID {
		return platformerrors.Validation("invalid parent department", map[string]string{
			"parent_id": "department cannot be its own parent",
		})
	}

	if d.CompleteName == "" {
		d.CompleteName = d.Name
	}

	return nil
}
