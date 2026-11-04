package accounting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type ListAccountsTreeUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewListAccountsTreeUseCase(repo AccountingRepository, logger *slog.Logger) *ListAccountsTreeUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &ListAccountsTreeUseCase{repo: repo, logger: logger}
}

func (uc *ListAccountsTreeUseCase) Execute(ctx context.Context) ([]AccountTreeNode, error) {
	logger := useCaseLogger(ctx, uc.logger, "list_accounts_tree")
	accounts, err := uc.repo.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}

	byID := make(map[uuid.UUID]*Account, len(accounts))
	children := make(map[uuid.UUID][]uuid.UUID, len(accounts))
	roots := make([]uuid.UUID, 0)
	for _, account := range accounts {
		if account == nil {
			return nil, errors.New("account repository returned a nil account")
		}
		if _, exists := byID[account.ID]; exists {
			return nil, fmt.Errorf("account repository returned duplicate id %s", account.ID)
		}
		byID[account.ID] = account
	}
	for _, account := range accounts {
		if account.ParentID == nil {
			roots = append(roots, account.ID)
			continue
		}
		if _, exists := byID[*account.ParentID]; !exists {
			return nil, fmt.Errorf("account %s references missing parent %s", account.ID, *account.ParentID)
		}
		children[*account.ParentID] = append(children[*account.ParentID], account.ID)
	}

	visitState := make(map[uuid.UUID]uint8, len(accounts))
	var visit func(uuid.UUID) error
	visit = func(id uuid.UUID) error {
		switch visitState[id] {
		case 1:
			return fmt.Errorf("account hierarchy contains a cycle at %s", id)
		case 2:
			return nil
		}
		visitState[id] = 1
		if parentID := byID[id].ParentID; parentID != nil {
			if err := visit(*parentID); err != nil {
				return err
			}
		}
		visitState[id] = 2
		return nil
	}
	for id := range byID {
		if err := visit(id); err != nil {
			return nil, err
		}
	}

	var build func(uuid.UUID) AccountTreeNode
	build = func(id uuid.UUID) AccountTreeNode {
		node := AccountTreeNode{AccountResponse: ToAccountResponse(byID[id])}
		for _, childID := range children[id] {
			node.Children = append(node.Children, build(childID))
		}
		return node
	}
	tree := make([]AccountTreeNode, 0, len(roots))
	for _, rootID := range roots {
		tree = append(tree, build(rootID))
	}
	logger.DebugContext(ctx, "account tree generated", slog.Int("account_count", len(accounts)))
	return tree, nil
}
