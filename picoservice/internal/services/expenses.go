package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/validation"
)

const descriptionMaxLength = 1000

// ProjectLookup loads a project in an organization. It is used to derive the
// expense currency and validate project ownership.
type ProjectLookup interface {
	FindByID(ctx context.Context, orgID, projectID string) (entities.Project, error)
}

// CreateExpenseInput contains the fields needed to create an expense. Currency
// is inherited from the project.
type CreateExpenseInput struct {
	OrgID       string
	UserID      string
	ProjectID   string
	Type        string
	Amount      int64
	Description string
	Tags        []string
	SpentAt     time.Time
}

// UpdateExpenseInput contains optional expense fields. A nil field is left
// unchanged. Currency cannot be changed.
type UpdateExpenseInput struct {
	OrgID       string
	ID          string
	ProjectID   *string
	Type        *string
	Amount      *int64
	Description *string
	Tags        *[]string
	SpentAt     *time.Time
}

// ListExpensesInput contains the expense list filters.
type ListExpensesInput struct {
	OrgID     string
	ProjectID string
	Tags      []string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

// ExpenseList contains one page of expenses and totals over every expense
// matching the filter.
type ExpenseList struct {
	Items       []entities.Expense
	TotalCount  int64
	TotalDebit  int64
	TotalCredit int64
	NetTotal    int64
}

// ExpenseService applies expense validation and business rules.
type ExpenseService struct {
	store    dao.ExpenseDAO
	projects ProjectLookup
}

// NewExpenseService returns an expense service backed by store and projects.
func NewExpenseService(store dao.ExpenseDAO, projects ProjectLookup) *ExpenseService {
	return &ExpenseService{store: store, projects: projects}
}

// Create validates an expense, checks that its project exists, and stores it.
func (s *ExpenseService) Create(ctx context.Context, in CreateExpenseInput) (entities.Expense, error) {
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return entities.Expense{}, &validation.Error{Field: "projectId", Message: "projectId is required"}
	}
	if in.Amount <= 0 {
		return entities.Expense{}, &validation.Error{Field: "amount", Message: "amount must be greater than zero"}
	}

	expenseType, err := normalizeExpenseType(in.Type, true)
	if err != nil {
		return entities.Expense{}, err
	}

	description := strings.TrimSpace(in.Description)
	if len(description) > descriptionMaxLength {
		return entities.Expense{}, &validation.Error{Field: "description", Message: "description must be at most 1000 characters"}
	}

	tags, err := normalizeTags(in.Tags)
	if err != nil {
		return entities.Expense{}, err
	}

	if in.SpentAt.IsZero() {
		return entities.Expense{}, &validation.Error{Field: "spentAt", Message: "spentAt is required"}
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	project, err := s.projects.FindByID(ctx, in.OrgID, projectID)
	if errors.Is(err, entities.ErrProjectNotFound) {
		return entities.Expense{}, &validation.Error{Field: "projectId", Message: "project not found"}
	}
	if err != nil {
		return entities.Expense{}, fmt.Errorf("get project: %w", err)
	}

	now := time.Now().UTC()
	expense := entities.Expense{
		OrgID:       in.OrgID,
		UserID:      in.UserID,
		ProjectID:   projectID,
		Type:        expenseType,
		Amount:      in.Amount,
		Currency:    project.Currency,
		Description: description,
		Tags:        tags,
		SpentAt:     in.SpentAt.UTC(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := s.store.Insert(ctx, expense)
	if err != nil {
		return entities.Expense{}, fmt.Errorf("create expense: %w", err)
	}
	return created, nil
}

// Get returns an expense scoped to orgID.
func (s *ExpenseService) Get(ctx context.Context, orgID, id string) (entities.Expense, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	expense, err := s.store.FindByID(ctx, orgID, id)
	if err != nil {
		return entities.Expense{}, fmt.Errorf("get expense: %w", err)
	}
	return expense, nil
}

// List returns a page of expenses scoped to orgID, plus totals over every
// expense matching the filters.
func (s *ExpenseService) List(ctx context.Context, in ListExpensesInput) (ExpenseList, error) {
	limit, offset, err := validation.Pagination(in.Limit, in.Offset)
	if err != nil {
		return ExpenseList{}, err
	}
	if in.From != nil && in.To != nil && in.From.After(*in.To) {
		return ExpenseList{}, &validation.Error{Field: "from", Message: "from must be before to"}
	}

	filter := dao.ExpenseFilter{
		OrgID:     in.OrgID,
		ProjectID: strings.TrimSpace(in.ProjectID),
		Tags:      normalizeTagFilter(in.Tags),
		From:      in.From,
		To:        in.To,
		Limit:     limit,
		Offset:    offset,
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	items, err := s.store.List(ctx, filter)
	if err != nil {
		return ExpenseList{}, fmt.Errorf("list expenses: %w", err)
	}
	summary, err := s.store.Summarize(ctx, filter)
	if err != nil {
		return ExpenseList{}, fmt.Errorf("summarize expenses: %w", err)
	}
	return ExpenseList{
		Items:       items,
		TotalCount:  summary.Count,
		TotalDebit:  summary.DebitAmount,
		TotalCredit: summary.CreditAmount,
		NetTotal:    summary.CreditAmount - summary.DebitAmount,
	}, nil
}

// Update applies partial changes to an expense.
func (s *ExpenseService) Update(ctx context.Context, in UpdateExpenseInput) (entities.Expense, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	fields := dao.ExpenseUpdate{UpdatedAt: time.Now().UTC()}
	changed := false

	if in.ProjectID != nil {
		projectID := strings.TrimSpace(*in.ProjectID)
		if projectID == "" {
			return entities.Expense{}, &validation.Error{Field: "projectId", Message: "projectId is required"}
		}

		project, err := s.projects.FindByID(ctx, in.OrgID, projectID)
		if errors.Is(err, entities.ErrProjectNotFound) {
			return entities.Expense{}, &validation.Error{Field: "projectId", Message: "project not found"}
		}
		if err != nil {
			return entities.Expense{}, fmt.Errorf("get project: %w", err)
		}
		_ = project

		fields.ProjectID = &projectID
		changed = true
	}

	if in.Amount != nil {
		if *in.Amount <= 0 {
			return entities.Expense{}, &validation.Error{Field: "amount", Message: "amount must be greater than zero"}
		}
		fields.Amount = in.Amount
		changed = true
	}

	if in.Type != nil {
		expenseType, err := normalizeExpenseType(*in.Type, false)
		if err != nil {
			return entities.Expense{}, err
		}
		fields.Type = &expenseType
		changed = true
	}

	if in.Description != nil {
		description := strings.TrimSpace(*in.Description)
		if len(description) > descriptionMaxLength {
			return entities.Expense{}, &validation.Error{Field: "description", Message: "description must be at most 1000 characters"}
		}
		fields.Description = &description
		changed = true
	}

	if in.Tags != nil {
		tags, err := normalizeTags(*in.Tags)
		if err != nil {
			return entities.Expense{}, err
		}
		fields.Tags = &tags
		changed = true
	}

	if in.SpentAt != nil {
		if in.SpentAt.IsZero() {
			return entities.Expense{}, &validation.Error{Field: "spentAt", Message: "spentAt is required"}
		}
		spentAt := in.SpentAt.UTC()
		fields.SpentAt = &spentAt
		changed = true
	}

	if !changed {
		return entities.Expense{}, &validation.Error{Field: "body", Message: "no fields to update"}
	}

	updated, err := s.store.Update(ctx, in.OrgID, in.ID, fields)
	if err != nil {
		return entities.Expense{}, fmt.Errorf("update expense: %w", err)
	}
	return updated, nil
}

// Delete removes an expense from an organization.
func (s *ExpenseService) Delete(ctx context.Context, orgID, id string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	if err := s.store.Delete(ctx, orgID, id); err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	return nil
}

func normalizeExpenseType(raw string, defaultDebit bool) (entities.ExpenseType, error) {
	value := entities.ExpenseType(strings.ToLower(strings.TrimSpace(raw)))
	if value == "" && defaultDebit {
		return entities.ExpenseTypeDebit, nil
	}
	if !value.Valid() {
		return "", &validation.Error{Field: "type", Message: "type must be debit or credit"}
	}
	return value, nil
}
