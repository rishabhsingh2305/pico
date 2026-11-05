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

// ProjectService is the project behavior used by the HTTP controllers.
type ProjectService interface {
	Create(ctx context.Context, in services.CreateProjectInput) (entities.Project, error)
	List(ctx context.Context, in services.ListProjectsInput) ([]entities.Project, error)
	Get(ctx context.Context, orgID, id string) (entities.Project, error)
	Update(ctx context.Context, in services.UpdateProjectInput) (entities.Project, error)
}

// ProjectController serves project endpoints.
type ProjectController struct {
	logger  *slog.Logger
	service ProjectService
}

// NewProjectController returns a project controller backed by service.
func NewProjectController(logger *slog.Logger, service ProjectService) *ProjectController {
	if logger == nil {
		logger = slog.Default()
	}
	return &ProjectController{logger: logger, service: service}
}

// Create handles POST /projects.
func (c *ProjectController) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	var request createProjectRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	project, err := c.service.Create(r.Context(), services.CreateProjectInput{
		OrgID:    id.OrgID,
		UserID:   id.UserID,
		Name:     request.Name,
		Currency: request.Currency,
		Tags:     request.Tags,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	w.Header().Set(httpx.HeaderLocation, "/projects/"+project.ID)
	httpx.WriteJSON(w, r, http.StatusCreated, projectResponseFrom(project))
}

// List handles GET and HEAD /projects.
func (c *ProjectController) List(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	limit, offset, message := parsePagination(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}

	items, err := c.service.List(r.Context(), services.ListProjectsInput{
		OrgID:  id.OrgID,
		Tags:   parseTags(r.URL.Query()),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	response := projectListResponse{Items: make([]projectResponse, 0, len(items))}
	for _, project := range items {
		response.Items = append(response.Items, projectResponseFrom(project))
	}
	httpx.WriteJSON(w, r, http.StatusOK, response)
}

// Get handles GET and HEAD /projects/{projectID}.
func (c *ProjectController) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	project, err := c.service.Get(r.Context(), id.OrgID, chi.URLParam(r, "projectID"))
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, projectResponseFrom(project))
}

// Update handles PATCH /projects/{projectID}.
func (c *ProjectController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	var request updateProjectRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	project, err := c.service.Update(r.Context(), services.UpdateProjectInput{
		OrgID: id.OrgID,
		ID:    chi.URLParam(r, "projectID"),
		Name:  request.Name,
		Tags:  request.Tags,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, projectResponseFrom(project))
}

func (c *ProjectController) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *validation.Error
	switch {
	case errors.As(err, &validationErr):
		httpx.WriteError(w, r, http.StatusBadRequest, validationErr.Message)
	case errors.Is(err, entities.ErrProjectNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "project not found")
	default:
		logInternalError(c.logger, w, r, "project request failed", err)
	}
}

type createProjectRequest struct {
	Name     string   `json:"name"`
	Currency string   `json:"currency"`
	Tags     []string `json:"tags"`
}

type updateProjectRequest struct {
	Name *string   `json:"name"`
	Tags *[]string `json:"tags"`
}

type projectResponse struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"orgId"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	Currency  string    `json:"currency"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type projectListResponse struct {
	Items []projectResponse `json:"items"`
}

func projectResponseFrom(project entities.Project) projectResponse {
	tags := project.Tags
	if tags == nil {
		tags = []string{}
	}

	return projectResponse{
		ID:        project.ID,
		OrgID:     project.OrgID,
		UserID:    project.UserID,
		Name:      project.Name,
		Currency:  project.Currency,
		Tags:      tags,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}
}
