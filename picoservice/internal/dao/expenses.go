package dao

import (
	"context"
	"time"

	"picoservice/internal/entities"
)

// ExpenseFilter contains the expense list filters.
type ExpenseFilter struct {
	OrgID      string
	ProjectIDs []string
	Tags       []string
	From       *time.Time
	To         *time.Time
	Limit      int
	Offset     int
}

// ExpenseSummaryGroup contains totals for one project and currency.
type ExpenseSummaryGroup struct {
	ProjectID    string
	Currency     string
	Count        int64
	DebitAmount  int64
	CreditAmount int64
}

// ExpenseUpdate contains normalized fields for a store update. UpdatedAt is
// always set.
type ExpenseUpdate struct {
	ProjectID   *string
	Type        *entities.ExpenseType
	Amount      *int64
	Description *string
	Tags        *[]string
	SpentAt     *time.Time
	UpdatedAt   time.Time
}

// ExpenseDAO persists expenses.
type ExpenseDAO interface {
	Insert(ctx context.Context, expense entities.Expense) (entities.Expense, error)
	FindByID(ctx context.Context, orgID, id string) (entities.Expense, error)
	List(ctx context.Context, filter ExpenseFilter) ([]entities.Expense, error)
	SummarizeByProject(ctx context.Context, filter ExpenseFilter) ([]ExpenseSummaryGroup, error)
	Update(ctx context.Context, orgID, id string, update ExpenseUpdate) (entities.Expense, error)
	Delete(ctx context.Context, orgID, id string) error
}
