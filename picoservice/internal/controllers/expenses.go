package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/entities"
	"picoservice/internal/httpx"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

// ExpenseService is the expense behavior used by the HTTP controllers.
type ExpenseService interface {
	Create(ctx context.Context, in services.CreateExpenseInput) (entities.Expense, error)
	List(ctx context.Context, in services.ListExpensesInput) (services.ExpenseList, error)
	Get(ctx context.Context, orgID, id string) (entities.Expense, error)
	Update(ctx context.Context, in services.UpdateExpenseInput) (entities.Expense, error)
	Delete(ctx context.Context, orgID, id string) error
}

// ExpenseController serves expense endpoints.
type ExpenseController struct {
	logger  *slog.Logger
	service ExpenseService
}

// NewExpenseController returns an expense controller backed by service.
func NewExpenseController(logger *slog.Logger, service ExpenseService) *ExpenseController {
	if logger == nil {
		logger = slog.Default()
	}
	return &ExpenseController{logger: logger, service: service}
}

// Create handles POST /expenses.
func (c *ExpenseController) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	var request createExpenseRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	expense, err := c.service.Create(r.Context(), services.CreateExpenseInput{
		OrgID:       id.OrgID,
		UserID:      id.UserID,
		ProjectID:   request.ProjectID,
		Type:        request.Type,
		Amount:      request.Amount,
		Description: request.Description,
		Tags:        request.Tags,
		SpentAt:     request.SpentAt,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	w.Header().Set(httpx.HeaderLocation, "/expenses/"+expense.ID)
	httpx.WriteJSON(w, r, http.StatusCreated, expenseResponseFrom(expense))
}

// List handles GET and HEAD /expenses.
func (c *ExpenseController) List(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	limit, offset, message := parsePagination(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}

	from, message := parseTimeQuery(r, "from")
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}
	to, message := parseTimeQuery(r, "to")
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}

	query := r.URL.Query()
	result, err := c.service.List(r.Context(), services.ListExpensesInput{
		OrgID:     id.OrgID,
		ProjectID: query.Get("projectId"),
		Tags:      parseTags(query),
		From:      from,
		To:        to,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	response := expenseListResponse{
		Items:       make([]expenseResponse, 0, len(result.Items)),
		TotalCount:  result.TotalCount,
		TotalDebit:  result.TotalDebit,
		TotalCredit: result.TotalCredit,
		NetTotal:    result.NetTotal,
	}
	for _, expense := range result.Items {
		response.Items = append(response.Items, expenseResponseFrom(expense))
	}
	httpx.WriteJSON(w, r, http.StatusOK, response)
}

// Get handles GET and HEAD /expenses/{expenseID}.
func (c *ExpenseController) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	expense, err := c.service.Get(r.Context(), id.OrgID, chi.URLParam(r, "expenseID"))
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, expenseResponseFrom(expense))
}

// Update handles PATCH /expenses/{expenseID}.
func (c *ExpenseController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	var request updateExpenseRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	expense, err := c.service.Update(r.Context(), services.UpdateExpenseInput{
		OrgID:       id.OrgID,
		ID:          chi.URLParam(r, "expenseID"),
		ProjectID:   request.ProjectID,
		Type:        request.Type,
		Amount:      request.Amount,
		Description: request.Description,
		Tags:        request.Tags,
		SpentAt:     request.SpentAt,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, expenseResponseFrom(expense))
}

// Delete handles DELETE /expenses/{expenseID}.
func (c *ExpenseController) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	if err := c.service.Delete(r.Context(), id.OrgID, chi.URLParam(r, "expenseID")); err != nil {
		c.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *ExpenseController) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *validation.Error
	switch {
	case errors.As(err, &validationErr):
		httpx.WriteError(w, r, http.StatusBadRequest, validationErr.Message)
	case errors.Is(err, entities.ErrExpenseNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "expense not found")
	default:
		logInternalError(c.logger, w, r, "expense request failed", err)
	}
}

func parseTimeQuery(r *http.Request, key string) (*time.Time, string) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, ""
	}

	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, key + " must be an RFC3339 timestamp"
	}
	return &value, ""
}

type createExpenseRequest struct {
	ProjectID   string    `json:"projectId"`
	Type        string    `json:"type"`
	Amount      int64     `json:"amount"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	SpentAt     time.Time `json:"spentAt"`
}

type updateExpenseRequest struct {
	ProjectID   *string    `json:"projectId"`
	Type        *string    `json:"type"`
	Amount      *int64     `json:"amount"`
	Description *string    `json:"description"`
	Tags        *[]string  `json:"tags"`
	SpentAt     *time.Time `json:"spentAt"`
}

type expenseResponse struct {
	ID          string               `json:"id"`
	OrgID       string               `json:"orgId"`
	UserID      string               `json:"userId"`
	ProjectID   string               `json:"projectId"`
	Type        entities.ExpenseType `json:"type"`
	Amount      int64                `json:"amount"`
	Currency    string               `json:"currency"`
	Description string               `json:"description"`
	Tags        []string             `json:"tags"`
	SpentAt     time.Time            `json:"spentAt"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
}

type expenseListResponse struct {
	Items       []expenseResponse `json:"items"`
	TotalCount  int64             `json:"totalCount"`
	TotalDebit  int64             `json:"totalDebit"`
	TotalCredit int64             `json:"totalCredit"`
	NetTotal    int64             `json:"netTotal"`
}

func expenseResponseFrom(expense entities.Expense) expenseResponse {
	tags := expense.Tags
	if tags == nil {
		tags = []string{}
	}

	return expenseResponse{
		ID:          expense.ID,
		OrgID:       expense.OrgID,
		UserID:      expense.UserID,
		ProjectID:   expense.ProjectID,
		Type:        expense.Type.OrDefault(),
		Amount:      expense.Amount,
		Currency:    expense.Currency,
		Description: expense.Description,
		Tags:        tags,
		SpentAt:     expense.SpentAt,
		CreatedAt:   expense.CreatedAt,
		UpdatedAt:   expense.UpdatedAt,
	}
}
