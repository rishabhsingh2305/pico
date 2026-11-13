package expensestest

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"picoservice/internal/dao"
	"picoservice/internal/entities"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

func TestCreateNormalizesFields(t *testing.T) {
	store := &fakeStore{}
	projects := &fakeProjectChecker{exists: true}
	service := services.NewExpenseService(store, projects)

	expense, err := service.Create(context.Background(), services.CreateExpenseInput{
		OrgID:       "org-1",
		UserID:      "user-1",
		ProjectID:   "project-1",
		Type:        "credit",
		Amount:      1299,
		Description: "  Lunch  ",
		Tags:        []string{" food ", "travel", "food"},
		SpentAt:     time.Date(2026, 1, 2, 12, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}

	if got, want := expense.Type, entities.ExpenseTypeCredit; got != want {
		t.Fatalf("expected type %q, got %q", want, got)
	}
	if got, want := expense.Currency, "USD"; got != want {
		t.Fatalf("expected currency %q, got %q", want, got)
	}
	if got, want := expense.Description, "Lunch"; got != want {
		t.Fatalf("expected description %q, got %q", want, got)
	}
	if got, want := len(expense.Tags), 2; got != want {
		t.Fatalf("expected %d tags, got %d", want, got)
	}
	if got, want := expense.Tags[0], "food"; got != want {
		t.Fatalf("expected first tag %q, got %q", want, got)
	}
	if got, want := expense.Tags[1], "travel"; got != want {
		t.Fatalf("expected second tag %q, got %q", want, got)
	}
	if projects.lastOrgID != "org-1" || projects.lastProjectID != "project-1" {
		t.Fatalf("expected project check for org/project, got %q/%q", projects.lastOrgID, projects.lastProjectID)
	}
}

func TestCreateDefaultsToDebit(t *testing.T) {
	service := services.NewExpenseService(&fakeStore{}, &fakeProjectChecker{exists: true})

	expense, err := service.Create(context.Background(), services.CreateExpenseInput{
		ProjectID: "project-1",
		Amount:    100,
		SpentAt:   time.Now(),
	})
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}
	if got, want := expense.Type, entities.ExpenseTypeDebit; got != want {
		t.Fatalf("expected type %q, got %q", want, got)
	}
}

func TestCreateValidation(t *testing.T) {
	tests := []struct {
		name  string
		input services.CreateExpenseInput
	}{
		{
			name:  "missing project",
			input: services.CreateExpenseInput{Amount: 1, SpentAt: time.Now()},
		},
		{
			name:  "zero amount",
			input: services.CreateExpenseInput{ProjectID: "project-1", SpentAt: time.Now()},
		},
		{
			name:  "invalid type",
			input: services.CreateExpenseInput{ProjectID: "project-1", Type: "transfer", Amount: 1, SpentAt: time.Now()},
		},
		{
			name:  "missing spent at",
			input: services.CreateExpenseInput{ProjectID: "project-1", Amount: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := services.NewExpenseService(&fakeStore{}, &fakeProjectChecker{exists: true})

			_, err := service.Create(context.Background(), tt.input)
			var validationErr *validation.Error
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestCreateRequiresProject(t *testing.T) {
	service := services.NewExpenseService(&fakeStore{}, &fakeProjectChecker{exists: false})

	_, err := service.Create(context.Background(), services.CreateExpenseInput{
		ProjectID: "project-1",
		Amount:    1,
		SpentAt:   time.Now(),
	})
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestListRejectsInvertedRange(t *testing.T) {
	service := services.NewExpenseService(&fakeStore{}, &fakeProjectChecker{})
	from := time.Now()
	to := from.Add(-time.Hour)

	_, err := service.List(context.Background(), services.ListExpensesInput{
		OrgID: "org-1",
		From:  &from,
		To:    &to,
	})
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestListReturnsNextOffsetAndNormalizesTags(t *testing.T) {
	store := &fakeStore{
		items: []entities.Expense{
			{ID: "expense-1"},
			{ID: "expense-2"},
		},
	}
	service := services.NewExpenseService(store, &fakeProjectChecker{})

	result, err := service.List(context.Background(), services.ListExpensesInput{
		OrgID: "org-1",
		Tags:  []string{" food ", "food", "travel"},
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("list expenses: %v", err)
	}
	if got, want := len(result.Items), 1; got != want {
		t.Fatalf("expected %d item, got %d", want, got)
	}
	if result.NextOffset == nil || *result.NextOffset != 1 {
		t.Fatalf("expected next offset 1, got %v", result.NextOffset)
	}
	if got, want := store.lastFilter.Limit, 2; got != want {
		t.Fatalf("expected requested limit %d, got %d", want, got)
	}
	if got, want := len(store.lastFilter.Tags), 2; got != want {
		t.Fatalf("expected %d normalized tags, got %v", want, store.lastFilter.Tags)
	}
	if got, want := store.lastFilter.Tags[0], "food"; got != want {
		t.Fatalf("expected first tag %q, got %q", want, got)
	}
	if got, want := store.lastFilter.Tags[1], "travel"; got != want {
		t.Fatalf("expected second tag %q, got %q", want, got)
	}
}

func TestListOmitsNextOffsetOnLastPage(t *testing.T) {
	store := &fakeStore{items: []entities.Expense{{ID: "expense-1"}}}
	service := services.NewExpenseService(store, &fakeProjectChecker{})

	result, err := service.List(context.Background(), services.ListExpensesInput{
		OrgID: "org-1",
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("list expenses: %v", err)
	}
	if result.NextOffset != nil {
		t.Fatalf("expected no next offset, got %v", *result.NextOffset)
	}
}

func TestSummarizeNormalizesFilters(t *testing.T) {
	store := &fakeStore{
		summary: []dao.ExpenseSummaryGroup{
			{ProjectID: "project-1", Currency: "USD", Count: 2, DebitAmount: 300, CreditAmount: 100},
		},
	}
	service := services.NewExpenseService(store, &fakeProjectChecker{})

	result, err := service.Summarize(context.Background(), services.SummarizeExpensesInput{
		OrgID:      "org-1",
		ProjectIDs: []string{" project-1 "},
		Tags:       []string{" food ", "food", "travel"},
	})
	if err != nil {
		t.Fatalf("summarize expenses: %v", err)
	}
	if got, want := len(result), 1; got != want {
		t.Fatalf("expected %d group, got %d", want, got)
	}
	if got, want := result[0].ProjectID, "project-1"; got != want {
		t.Fatalf("expected project %q, got %q", want, got)
	}
	if got, want := store.lastFilter.ProjectIDs, []string{"project-1"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("expected filter projects %v, got %v", want, got)
	}
	if got, want := len(store.lastFilter.Tags), 2; got != want {
		t.Fatalf("expected %d normalized tags, got %v", want, store.lastFilter.Tags)
	}
}

func TestSummarizeRejectsTooManyProjectIDs(t *testing.T) {
	projectIDs := make([]string, 0, 101)
	for index := 0; index < 101; index++ {
		projectIDs = append(projectIDs, "project-"+strconv.Itoa(index))
	}
	service := services.NewExpenseService(&fakeStore{}, &fakeProjectChecker{})

	_, err := service.Summarize(context.Background(), services.SummarizeExpensesInput{
		OrgID:      "org-1",
		ProjectIDs: projectIDs,
	})
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestSummarizeRejectsInvertedRange(t *testing.T) {
	service := services.NewExpenseService(&fakeStore{}, &fakeProjectChecker{})
	from := time.Now()
	to := from.Add(-time.Hour)

	_, err := service.Summarize(context.Background(), services.SummarizeExpensesInput{
		OrgID: "org-1",
		From:  &from,
		To:    &to,
	})
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpdateRequiresFields(t *testing.T) {
	service := services.NewExpenseService(&fakeStore{}, &fakeProjectChecker{})

	_, err := service.Update(context.Background(), services.UpdateExpenseInput{OrgID: "org-1", ID: "expense-1"})
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpdateNormalizesFields(t *testing.T) {
	store := &fakeStore{}
	projects := &fakeProjectChecker{exists: true}
	service := services.NewExpenseService(store, projects)

	projectID := "project-2"
	expenseType := "credit"
	amount := int64(500)
	tags := []string{"b", "a", "b"}
	spentAt := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)

	_, err := service.Update(context.Background(), services.UpdateExpenseInput{
		OrgID:     "org-1",
		ID:        "expense-1",
		ProjectID: &projectID,
		Type:      &expenseType,
		Amount:    &amount,
		Tags:      &tags,
		SpentAt:   &spentAt,
	})
	if err != nil {
		t.Fatalf("update expense: %v", err)
	}

	if store.lastUpdate.Type == nil || *store.lastUpdate.Type != entities.ExpenseTypeCredit {
		t.Fatalf("expected type credit, got %v", store.lastUpdate.Type)
	}
	if store.lastUpdate.Tags == nil || len(*store.lastUpdate.Tags) != 2 {
		t.Fatalf("expected two normalized tags, got %v", store.lastUpdate.Tags)
	}
	if got, want := (*store.lastUpdate.Tags)[0], "a"; got != want {
		t.Fatalf("expected first tag %q, got %q", want, got)
	}
}

func TestUpdateRejectsInvalidType(t *testing.T) {
	service := services.NewExpenseService(&fakeStore{}, &fakeProjectChecker{})
	expenseType := "transfer"

	_, err := service.Update(context.Background(), services.UpdateExpenseInput{
		OrgID: "org-1",
		ID:    "expense-1",
		Type:  &expenseType,
	})
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpdateRequiresProject(t *testing.T) {
	service := services.NewExpenseService(&fakeStore{}, &fakeProjectChecker{exists: false})
	projectID := "project-1"

	_, err := service.Update(context.Background(), services.UpdateExpenseInput{
		OrgID:     "org-1",
		ID:        "expense-1",
		ProjectID: &projectID,
	})
	var validationErr *validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	store := &fakeStore{}
	service := services.NewExpenseService(store, &fakeProjectChecker{})

	if err := service.Delete(context.Background(), "org-1", "expense-1"); err != nil {
		t.Fatalf("delete expense: %v", err)
	}
	if store.deletedOrgID != "org-1" || store.deletedID != "expense-1" {
		t.Fatalf("expected delete call, got %q/%q", store.deletedOrgID, store.deletedID)
	}
}

type fakeStore struct {
	deletedOrgID string
	deletedID    string
	lastUpdate   dao.ExpenseUpdate
	lastFilter   dao.ExpenseFilter
	items        []entities.Expense
	summary      []dao.ExpenseSummaryGroup
}

func (s *fakeStore) Insert(_ context.Context, expense entities.Expense) (entities.Expense, error) {
	expense.ID = "expense-1"
	return expense, nil
}

func (s *fakeStore) FindByID(_ context.Context, _, _ string) (entities.Expense, error) {
	return entities.Expense{}, entities.ErrExpenseNotFound
}

func (s *fakeStore) List(_ context.Context, filter dao.ExpenseFilter) ([]entities.Expense, error) {
	s.lastFilter = filter
	return s.items, nil
}

func (s *fakeStore) SummarizeByProject(_ context.Context, filter dao.ExpenseFilter) ([]dao.ExpenseSummaryGroup, error) {
	s.lastFilter = filter
	return s.summary, nil
}

func (s *fakeStore) Update(_ context.Context, _, _ string, update dao.ExpenseUpdate) (entities.Expense, error) {
	s.lastUpdate = update
	return entities.Expense{}, nil
}

func (s *fakeStore) Delete(_ context.Context, orgID, id string) error {
	s.deletedOrgID = orgID
	s.deletedID = id
	return nil
}

type fakeProjectChecker struct {
	exists        bool
	lastOrgID     string
	lastProjectID string
}

func (c *fakeProjectChecker) FindByID(_ context.Context, orgID, projectID string) (entities.Project, error) {
	c.lastOrgID = orgID
	c.lastProjectID = projectID
	if !c.exists {
		return entities.Project{}, entities.ErrProjectNotFound
	}
	return entities.Project{ID: projectID, OrgID: orgID, Currency: "USD"}, nil
}
