package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/entities"
	"picoservice/internal/httpx"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

// CredentialService is the share behavior used by the HTTP controllers.
type CredentialService interface {
	Create(ctx context.Context, in services.CreateCredentialInput) (services.CreateCredentialResult, error)
	List(ctx context.Context, in services.ListCredentialsInput) ([]entities.Credential, error)
	Revoke(ctx context.Context, orgID, id string) error
}

// CredentialController serves share management endpoints.
type CredentialController struct {
	logger  *slog.Logger
	service CredentialService
}

// NewCredentialController returns a credential controller.
func NewCredentialController(logger *slog.Logger, service CredentialService) *CredentialController {
	if logger == nil {
		logger = slog.Default()
	}
	return &CredentialController{logger: logger, service: service}
}

// Create handles POST /orgs/{orgID}/shares.
func (c *CredentialController) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	var request createShareRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := c.service.Create(r.Context(), services.CreateCredentialInput{
		ActorID:      id.UserID,
		OrgID:        chi.URLParam(r, "orgID"),
		Kind:         entities.CredentialKindShare,
		ResourceType: request.ResourceType,
		ResourceID:   request.ResourceID,
		Name:         request.Name,
		ExpiresAt:    request.ExpiresAt,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	w.Header().Set(httpx.HeaderLocation, "/orgs/"+result.Credential.OrgID+"/shares/"+result.Credential.ID)
	httpx.WriteJSON(w, r, http.StatusCreated, shareResponseFrom(result))
}

// List handles GET /orgs/{orgID}/shares.
func (c *CredentialController) List(w http.ResponseWriter, r *http.Request) {
	c.listFromQuery(w, r, entities.CredentialKindShare, false)
}

// Credentials handles GET /orgs/{orgID}/credentials.
func (c *CredentialController) Credentials(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	if kind == "" {
		kind = entities.CredentialKindShare
	}
	c.listFromQuery(w, r, kind, true)
}

func (c *CredentialController) listFromQuery(w http.ResponseWriter, r *http.Request, kind string, allowFilters bool) {
	limit, offset, message := parsePagination(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}

	in := services.ListCredentialsInput{
		OrgID:  chi.URLParam(r, "orgID"),
		Kind:   kind,
		Limit:  limit,
		Offset: offset,
	}
	if allowFilters {
		in.ResourceType = r.URL.Query().Get("resourceType")
		in.ResourceIDs = parseIDList(r.URL.Query()["resourceIds"])
		in.Active = r.URL.Query().Get("active") == "true"
	}

	items, err := c.service.List(r.Context(), in)
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	response := credentialListResponse{Items: make([]credentialResponse, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, credentialResponseFrom(item))
	}
	httpx.WriteJSON(w, r, http.StatusOK, response)
}

// Status handles GET /orgs/{orgID}/shares/status.
func (c *CredentialController) Status(w http.ResponseWriter, r *http.Request) {
	resourceType := strings.TrimSpace(r.URL.Query().Get("resourceType"))
	if resourceType == "" {
		httpx.WriteError(w, r, http.StatusBadRequest, "resourceType is required")
		return
	}
	resourceIDs := parseIDList(r.URL.Query()["resourceIds"])
	if len(resourceIDs) == 0 {
		httpx.WriteError(w, r, http.StatusBadRequest, "resourceIds is required")
		return
	}

	items, err := c.service.List(r.Context(), services.ListCredentialsInput{
		OrgID:        chi.URLParam(r, "orgID"),
		Kind:         entities.CredentialKindShare,
		ResourceType: resourceType,
		ResourceIDs:  resourceIDs,
		Active:       true,
		Limit:        len(resourceIDs),
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	type status struct {
		ResourceID  string     `json:"resourceId"`
		ActiveCount int        `json:"activeCount"`
		ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	}
	byID := make(map[string]*status, len(items))
	for _, item := range items {
		entry, ok := byID[item.ResourceID]
		if !ok {
			entry = &status{ResourceID: item.ResourceID}
			byID[item.ResourceID] = entry
		}
		entry.ActiveCount++
		if item.ExpiresAt != nil && (entry.ExpiresAt == nil || item.ExpiresAt.Before(*entry.ExpiresAt)) {
			entry.ExpiresAt = item.ExpiresAt
		}
	}

	ordered := make([]*status, 0, len(byID))
	for _, id := range resourceIDs {
		if entry, ok := byID[id]; ok {
			ordered = append(ordered, entry)
		}
	}
	httpx.WriteJSON(w, r, http.StatusOK, map[string]any{
		"resourceType": resourceType,
		"items":        ordered,
	})
}

// Revoke handles DELETE /orgs/{orgID}/shares/{shareID}.
func (c *CredentialController) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := c.service.Revoke(r.Context(), chi.URLParam(r, "orgID"), chi.URLParam(r, "shareID")); err != nil {
		c.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *CredentialController) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *validation.Error
	switch {
	case errors.As(err, &validationErr):
		httpx.WriteError(w, r, http.StatusBadRequest, validationErr.Message)
	case errors.Is(err, entities.ErrOrgNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "org not found")
	case errors.Is(err, entities.ErrProjectNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "project not found")
	case errors.Is(err, entities.ErrExpenseNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "expense not found")
	case errors.Is(err, entities.ErrCredentialNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "share not found")
	default:
		logInternalError(c.logger, w, r, "credential request failed", err)
	}
}

type createShareRequest struct {
	ResourceType string     `json:"resourceType"`
	ResourceID   string     `json:"resourceId"`
	Name         string     `json:"name"`
	ExpiresAt    *time.Time `json:"expiresAt"`
}

type credentialResponse struct {
	ID           string     `json:"id"`
	Kind         string     `json:"kind"`
	OrgID        string     `json:"orgId"`
	ResourceType string     `json:"resourceType,omitempty"`
	ResourceID   string     `json:"resourceId,omitempty"`
	ProjectID    string     `json:"projectId,omitempty"`
	Scopes       []string   `json:"scopes"`
	Name         string     `json:"name,omitempty"`
	CreatedBy    string     `json:"createdBy"`
	CreatedAt    time.Time  `json:"createdAt"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	RevokedAt    *time.Time `json:"revokedAt,omitempty"`
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
	Status       string     `json:"status"`
}

type shareResponse struct {
	credentialResponse
	Token     string `json:"token"`
	SharePath string `json:"sharePath"`
}

type credentialListResponse struct {
	Items []credentialResponse `json:"items"`
}

func credentialResponseFrom(credential entities.Credential) credentialResponse {
	return credentialResponse{
		ID:           credential.ID,
		Kind:         credential.Kind,
		OrgID:        credential.OrgID,
		ResourceType: credential.ResourceType,
		ResourceID:   credential.ResourceID,
		ProjectID:    credential.ProjectID,
		Scopes:       credential.Scopes,
		Name:         credential.Name,
		CreatedBy:    credential.CreatedBy,
		CreatedAt:    credential.CreatedAt,
		ExpiresAt:    credential.ExpiresAt,
		RevokedAt:    credential.RevokedAt,
		LastUsedAt:   credential.LastUsedAt,
		Status:       credential.Status(time.Now().UTC()),
	}
}

func shareResponseFrom(result services.CreateCredentialResult) shareResponse {
	return shareResponse{
		credentialResponse: credentialResponseFrom(result.Credential),
		Token:              result.Token,
		SharePath:          sharePath(result.Credential, result.Token),
	}
}

func sharePath(credential entities.Credential, token string) string {
	values := url.Values{}
	values.Set("orgId", credential.OrgID)
	values.Set("resourceType", credential.ResourceType)
	values.Set("resourceId", credential.ResourceID)
	if credential.ProjectID != "" {
		values.Set("projectId", credential.ProjectID)
	}
	return "/shared?" + values.Encode() + "#share=" + url.QueryEscape(token)
}

func parseIDList(values []string) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0, len(values))
	for _, value := range values {
		for _, id := range strings.Split(value, ",") {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}
