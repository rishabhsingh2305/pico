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

// MembershipService is the membership behavior used by the HTTP controllers.
type MembershipService interface {
	Add(ctx context.Context, in services.CreateMembershipInput) (entities.Membership, error)
	List(ctx context.Context, in services.ListMembershipsInput) ([]entities.Membership, error)
	UpdateRoles(ctx context.Context, orgID, userID string, roles map[string]string) error
	Remove(ctx context.Context, orgID, userID string) error
}

// MembershipController serves organization membership endpoints.
type MembershipController struct {
	logger  *slog.Logger
	service MembershipService
}

// NewMembershipController returns a membership controller backed by service.
func NewMembershipController(logger *slog.Logger, service MembershipService) *MembershipController {
	if logger == nil {
		logger = slog.Default()
	}
	return &MembershipController{logger: logger, service: service}
}

// Add handles POST /orgs/{orgID}/members.
func (c *MembershipController) Add(w http.ResponseWriter, r *http.Request) {
	var request addMemberRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	membership, err := c.service.Add(r.Context(), services.CreateMembershipInput{
		OrgID:  chi.URLParam(r, "orgID"),
		UserID: request.UserID,
		Roles:  request.Roles,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	w.Header().Set(httpx.HeaderLocation, "/orgs/"+membership.OrgID+"/members/"+membership.UserID)
	httpx.WriteJSON(w, r, http.StatusCreated, membershipResponseFrom(membership))
}

// List handles GET and HEAD /orgs/{orgID}/members.
func (c *MembershipController) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, message := parsePagination(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}

	items, err := c.service.List(r.Context(), services.ListMembershipsInput{
		OrgID:  chi.URLParam(r, "orgID"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	response := membershipListResponse{Items: make([]membershipResponse, 0, len(items))}
	for _, membership := range items {
		response.Items = append(response.Items, membershipResponseFrom(membership))
	}
	httpx.WriteJSON(w, r, http.StatusOK, response)
}

// UpdateRoles handles PATCH /orgs/{orgID}/members/{memberUserID}.
func (c *MembershipController) UpdateRoles(w http.ResponseWriter, r *http.Request) {
	var request updateMemberRolesRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := c.service.UpdateRoles(
		r.Context(),
		chi.URLParam(r, "orgID"),
		chi.URLParam(r, "memberUserID"),
		request.Roles,
	); err != nil {
		c.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Remove handles DELETE /orgs/{orgID}/members/{memberUserID}.
func (c *MembershipController) Remove(w http.ResponseWriter, r *http.Request) {
	if err := c.service.Remove(
		r.Context(),
		chi.URLParam(r, "orgID"),
		chi.URLParam(r, "memberUserID"),
	); err != nil {
		c.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *MembershipController) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *validation.Error
	switch {
	case errors.As(err, &validationErr):
		httpx.WriteError(w, r, http.StatusBadRequest, validationErr.Message)
	case errors.Is(err, entities.ErrOrgNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "org not found")
	case errors.Is(err, entities.ErrUserNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "user not found")
	case errors.Is(err, entities.ErrMembershipNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "membership not found")
	case errors.Is(err, entities.ErrAlreadyMember):
		httpx.WriteError(w, r, http.StatusConflict, "user is already a member")
	case errors.Is(err, entities.ErrCannotRemoveOwner):
		httpx.WriteError(w, r, http.StatusConflict, "cannot remove org owner")
	case errors.Is(err, entities.ErrCannotChangeOwner):
		httpx.WriteError(w, r, http.StatusConflict, "cannot change org owner role")
	default:
		logInternalError(c.logger, w, r, "membership request failed", err)
	}
}

type addMemberRequest struct {
	UserID string            `json:"userId"`
	Roles  map[string]string `json:"roles"`
}

type updateMemberRolesRequest struct {
	Roles map[string]string `json:"roles"`
}

type membershipResponse struct {
	ID        string            `json:"id"`
	OrgID     string            `json:"orgId"`
	UserID    string            `json:"userId"`
	Owner     bool              `json:"owner"`
	Roles     map[string]string `json:"roles,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
}

type membershipListResponse struct {
	Items []membershipResponse `json:"items"`
}

func membershipResponseFrom(membership entities.Membership) membershipResponse {
	roles := membership.Roles
	if membership.Owner {
		roles = nil
	}
	return membershipResponse{
		ID:        membership.ID,
		OrgID:     membership.OrgID,
		UserID:    membership.UserID,
		Owner:     membership.Owner,
		Roles:     roles,
		CreatedAt: membership.CreatedAt,
	}
}
