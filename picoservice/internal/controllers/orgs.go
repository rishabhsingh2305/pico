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

// OrgService is the organization behavior used by the HTTP controllers.
type OrgService interface {
	Create(ctx context.Context, in services.CreateOrgInput) (entities.Org, error)
	List(ctx context.Context, in services.ListOrgsInput) ([]entities.Org, error)
	Get(ctx context.Context, id string) (entities.Org, error)
	Update(ctx context.Context, in services.UpdateOrgInput) (entities.Org, error)
}

// OrgController serves organization endpoints.
type OrgController struct {
	logger  *slog.Logger
	service OrgService
}

// NewOrgController returns an organization controller backed by service.
func NewOrgController(logger *slog.Logger, service OrgService) *OrgController {
	if logger == nil {
		logger = slog.Default()
	}
	return &OrgController{logger: logger, service: service}
}

// Create handles POST /orgs.
func (c *OrgController) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	var request createOrgRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	org, err := c.service.Create(r.Context(), services.CreateOrgInput{
		ActorID: id.UserID,
		Name:    request.Name,
		Tags:    request.Tags,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	w.Header().Set(httpx.HeaderLocation, "/orgs/"+org.ID)
	httpx.WriteJSON(w, r, http.StatusCreated, orgResponseFrom(org))
}

// List handles GET and HEAD /orgs.
func (c *OrgController) List(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	limit, offset, message := parsePagination(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}

	items, err := c.service.List(r.Context(), services.ListOrgsInput{
		ActorID: id.UserID,
		Tags:    parseTags(r.URL.Query()),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	response := orgListResponse{Items: make([]orgResponse, 0, len(items))}
	for _, org := range items {
		response.Items = append(response.Items, orgResponseFrom(org))
	}
	httpx.WriteJSON(w, r, http.StatusOK, response)
}

// Get handles GET and HEAD /orgs/{orgID}.
func (c *OrgController) Get(w http.ResponseWriter, r *http.Request) {
	org, err := c.service.Get(r.Context(), chi.URLParam(r, "orgID"))
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, orgResponseFrom(org))
}

// Update handles PATCH /orgs/{orgID}.
func (c *OrgController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	var request updateOrgRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	org, err := c.service.Update(r.Context(), services.UpdateOrgInput{
		ActorID: id.UserID,
		OrgID:   chi.URLParam(r, "orgID"),
		Name:    request.Name,
		Tags:    request.Tags,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, orgResponseFrom(org))
}

func (c *OrgController) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *validation.Error
	switch {
	case errors.As(err, &validationErr):
		httpx.WriteError(w, r, http.StatusBadRequest, validationErr.Message)
	case errors.Is(err, entities.ErrOrgNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "org not found")
	case errors.Is(err, entities.ErrForbidden):
		httpx.WriteError(w, r, http.StatusForbidden, "forbidden")
	default:
		logInternalError(c.logger, w, r, "org request failed", err)
	}
}

type createOrgRequest struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type updateOrgRequest struct {
	Name *string   `json:"name"`
	Tags *[]string `json:"tags"`
}

type orgResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	UserID    string    `json:"userId"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type orgListResponse struct {
	Items []orgResponse `json:"items"`
}

func orgResponseFrom(org entities.Org) orgResponse {
	tags := org.Tags
	if tags == nil {
		tags = []string{}
	}

	return orgResponse{
		ID:        org.ID,
		Name:      org.Name,
		UserID:    org.UserID,
		Tags:      tags,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}
}
