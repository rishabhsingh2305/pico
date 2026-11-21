package controllers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"picoservice/internal/httpx"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

// LinkStatsService is the link analytics read behavior used by the controller.
type LinkStatsService interface {
	Stats(ctx context.Context, in services.LinkStatsInput) (services.LinkStats, error)
}

// LinkAnalyticsController serves link click analytics.
type LinkAnalyticsController struct {
	logger  *slog.Logger
	service LinkStatsService
}

// NewLinkAnalyticsController returns a controller backed by service.
func NewLinkAnalyticsController(logger *slog.Logger, service LinkStatsService) *LinkAnalyticsController {
	if logger == nil {
		logger = slog.Default()
	}
	return &LinkAnalyticsController{logger: logger, service: service}
}

// Stats handles GET and HEAD /orgs/{orgID}/analytics/links.
func (c *LinkAnalyticsController) Stats(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	linkIDs, message := parseLinkIDs(r)
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

	now := time.Now().UTC()
	if from == nil {
		value := now.Add(-30 * 24 * time.Hour)
		from = &value
	}
	if to == nil {
		to = &now
	}

	result, err := c.service.Stats(r.Context(), services.LinkStatsInput{
		OrgID:   id.OrgID,
		LinkIDs: linkIDs,
		From:    *from,
		To:      *to,
	})
	if err != nil {
		var validationErr *validation.Error
		switch {
		case errors.As(err, &validationErr):
			httpx.WriteError(w, r, http.StatusBadRequest, validationErr.Message)
		default:
			logInternalError(c.logger, w, r, "link stats request failed", err)
		}
		return
	}

	response := linkStatsResponse{
		Items: make([]linkStatsItem, 0, len(result.Items)),
	}
	for _, item := range result.Items {
		response.Items = append(response.Items, linkStatsItem{
			LinkID: item.LinkID,
			Clicks: item.Clicks,
		})
	}
	httpx.WriteJSON(w, r, http.StatusOK, response)
}

func parseLinkIDs(r *http.Request) ([]string, string) {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, value := range r.URL.Query()["ids"] {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			ids = append(ids, part)
		}
	}

	switch {
	case len(ids) == 0:
		return nil, "ids is required"
	case len(ids) > services.LinkStatsMaxIDs:
		return nil, fmt.Sprintf("ids must contain at most %d items", services.LinkStatsMaxIDs)
	default:
		return ids, ""
	}
}

type linkStatsItem struct {
	LinkID string `json:"linkId"`
	Clicks int64  `json:"clicks"`
}

type linkStatsResponse struct {
	Items []linkStatsItem `json:"items"`
}
