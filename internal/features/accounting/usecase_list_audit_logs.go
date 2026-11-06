package accounting

import (
	"context"
	"fmt"
)

type ListAuditLogsUseCase struct {
	repo AccountingRepository
}

func NewListAuditLogsUseCase(repo AccountingRepository) *ListAuditLogsUseCase {
	return &ListAuditLogsUseCase{repo: repo}
}

func (uc *ListAuditLogsUseCase) Execute(ctx context.Context, limit, offset int) (*ListAuditLogsResponse, error) {
	logs, total, err := uc.repo.ListFinancialAuditLogs(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list financial audit logs: %w", err)
	}
	responses := make([]FinancialAuditLogResponse, len(logs))
	for i, l := range logs {
		responses[i] = ToFinancialAuditLogResponse(l)
	}
	return &ListAuditLogsResponse{
		Logs:   responses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

