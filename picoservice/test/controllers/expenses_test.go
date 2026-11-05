package controllerstest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"picoservice/internal/controllers"
	"picoservice/internal/entities"
	"picoservice/internal/httpx"
	"picoservice/internal/services"
	"picoservice/internal/validation"
	"picoservice/test/fakes"
)

func TestExpenseCreate(t *testing.T) {
	service := &fakes.ExpenseService{
		CreateFunc: func(_ context.Context, in services.CreateExpenseInput) (entities.Expense, error) {
			if in.OrgID != "org-1" || in.UserID != "user-1" {
				t.Fatalf("unexpected identity: %q %q", in.OrgID, in.UserID)
			}
			if got, want := in.Type, "credit"; got != want {
				t.Fatalf("expected type %q, got %q", want, got)
			}
			return entities.Expense{
				ID:        "expense-1",
				OrgID:     in.OrgID,
				UserID:    in.UserID,
				ProjectID: in.ProjectID,
				Type:      entities.ExpenseTypeCredit,
				Amount:    in.Amount,
				Currency:  "USD",
				SpentAt:   in.SpentAt,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/expenses?orgId=org-1&userId=user-1",
		strings.NewReader(`{"projectId":"project-1","type":"credit","amount":100,"description":"Lunch","tags":[" food "],"spentAt":"2026-01-02T03:04:05Z"}`),
	)
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewExpenseController(testLogger(), service).Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if got, want := rec.Header().Get(httpx.HeaderLocation), "/expenses/expense-1"; got != want {
		t.Fatalf("expected Location %q, got %q", want, got)
	}
	var body struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body.Type, "credit"; got != want {
		t.Fatalf("expected type %q, got %q", want, got)
	}
}

func TestExpenseCreateValidation(t *testing.T) {
	service := &fakes.ExpenseService{
		CreateFunc: func(context.Context, services.CreateExpenseInput) (entities.Expense, error) {
			return entities.Expense{}, &validation.Error{Field: "amount", Message: "amount must be greater than zero"}
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/expenses?orgId=org-1&userId=user-1", strings.NewReader(`{"projectId":"project-1","amount":0,"spentAt":"2026-01-02T03:04:05Z"}`))
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewExpenseController(testLogger(), service).Create(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "amount must be greater than zero")
}

func TestExpenseListBadTime(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/expenses?orgId=org-1&userId=user-1&from=not-a-time", nil)
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewExpenseController(testLogger(), &fakes.ExpenseService{}).List(rec, req)

	assertJSONError(t, rec, http.StatusBadRequest, "from must be an RFC3339 timestamp")
}

func TestExpenseListSingleTagFallback(t *testing.T) {
	service := &fakes.ExpenseService{
		ListFunc: func(_ context.Context, in services.ListExpensesInput) (services.ExpenseList, error) {
			if got, want := in.Tags, []string{"travel"}; len(got) != 1 || got[0] != want[0] {
				t.Fatalf("expected tags %v, got %v", want, got)
			}
			return services.ExpenseList{Items: []entities.Expense{}}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/expenses?orgId=org-1&userId=user-1&tags=&tag=travel", nil)
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewExpenseController(testLogger(), service).List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestExpenseListResponse(t *testing.T) {
	service := &fakes.ExpenseService{
		ListFunc: func(_ context.Context, in services.ListExpensesInput) (services.ExpenseList, error) {
			if got, want := in.ProjectID, "project-1"; got != want {
				t.Fatalf("expected project %q, got %q", want, got)
			}
			if got, want := in.Tags, []string{"food", "travel"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
				t.Fatalf("expected tags %v, got %v", want, got)
			}
			return services.ExpenseList{
				Items: []entities.Expense{
					{ID: "expense-1", Currency: "USD", Type: entities.ExpenseTypeDebit, Amount: 1299},
				},
				TotalCount:  4,
				TotalDebit:  1000,
				TotalCredit: 9876,
				NetTotal:    8876,
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/expenses?orgId=org-1&userId=user-1&projectId=project-1&tags=food,travel&limit=1&offset=2", nil)
	req = withIdentity(req, "org-1", "user-1")

	controllers.NewExpenseController(testLogger(), service).List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	var body struct {
		Items       []map[string]any `json:"items"`
		TotalCount  int64            `json:"totalCount"`
		TotalDebit  int64            `json:"totalDebit"`
		TotalCredit int64            `json:"totalCredit"`
		NetTotal    int64            `json:"netTotal"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := len(body.Items), 1; got != want {
		t.Fatalf("expected %d item, got %d", want, got)
	}
	if got, want := body.Items[0]["type"], "debit"; got != want {
		t.Fatalf("expected item type %q, got %v", want, got)
	}
	if got, want := body.TotalCount, int64(4); got != want {
		t.Fatalf("expected total count %d, got %d", want, got)
	}
	if got, want := body.TotalDebit, int64(1000); got != want {
		t.Fatalf("expected total debit %d, got %d", want, got)
	}
	if got, want := body.TotalCredit, int64(9876); got != want {
		t.Fatalf("expected total credit %d, got %d", want, got)
	}
	if got, want := body.NetTotal, int64(8876); got != want {
		t.Fatalf("expected net total %d, got %d", want, got)
	}
}

func TestExpenseDelete(t *testing.T) {
	service := &fakes.ExpenseService{
		DeleteFunc: func(_ context.Context, orgID, id string) error {
			if orgID != "org-1" || id != "expense-1" {
				t.Fatalf("unexpected delete input: %q %q", orgID, id)
			}
			return nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/expenses/expense-1?orgId=org-1&userId=user-1", nil)
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"expenseID": "expense-1"})

	controllers.NewExpenseController(testLogger(), service).Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestExpenseUpdateNotFound(t *testing.T) {
	service := &fakes.ExpenseService{
		UpdateFunc: func(context.Context, services.UpdateExpenseInput) (entities.Expense, error) {
			return entities.Expense{}, entities.ErrExpenseNotFound
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/expenses/missing?orgId=org-1&userId=user-1", strings.NewReader(`{"amount":200}`))
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"expenseID": "missing"})

	controllers.NewExpenseController(testLogger(), service).Update(rec, req)

	assertJSONError(t, rec, http.StatusNotFound, "expense not found")
}

func TestExpenseUpdateForwardsType(t *testing.T) {
	service := &fakes.ExpenseService{
		UpdateFunc: func(_ context.Context, in services.UpdateExpenseInput) (entities.Expense, error) {
			if in.Type == nil || *in.Type != "credit" {
				t.Fatalf("expected type credit, got %v", in.Type)
			}
			return entities.Expense{ID: "expense-1", Type: entities.ExpenseTypeCredit}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/expenses/expense-1?orgId=org-1&userId=user-1", strings.NewReader(`{"type":"credit"}`))
	req = withIdentity(req, "org-1", "user-1")
	req = withRouteParams(req, map[string]string{"expenseID": "expense-1"})

	controllers.NewExpenseController(testLogger(), service).Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	var body struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body.Type, "credit"; got != want {
		t.Fatalf("expected type %q, got %q", want, got)
	}
}
