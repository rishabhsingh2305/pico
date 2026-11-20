package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/entities"
	"picoservice/internal/httpx"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

// LinkService is the link behavior used by the HTTP controllers.
type LinkService interface {
	Create(ctx context.Context, in services.CreateLinkInput) (entities.Link, error)
	List(ctx context.Context, in services.ListLinksInput) (services.LinkList, error)
	Get(ctx context.Context, orgID, id string) (entities.Link, error)
	Resolve(ctx context.Context, slug string) (entities.Link, error)
	Update(ctx context.Context, in services.UpdateLinkInput) (entities.Link, error)
	Delete(ctx context.Context, orgID, id string) (entities.Link, error)
}

// LinkPurger removes a public short link from the CDN cache.
type LinkPurger interface {
	Purge(ctx context.Context, url string) error
}

// LinkController serves short-link management and public redirects.
type LinkController struct {
	logger       *slog.Logger
	service      LinkService
	shortBaseURL string
	purger       LinkPurger
}

// NewLinkController returns a link controller backed by service.
func NewLinkController(logger *slog.Logger, service LinkService, shortBaseURL string, purger LinkPurger) *LinkController {
	if logger == nil {
		logger = slog.Default()
	}
	return &LinkController{
		logger:       logger,
		service:      service,
		shortBaseURL: strings.TrimRight(strings.TrimSpace(shortBaseURL), "/"),
		purger:       purger,
	}
}

// Create handles POST /orgs/{orgID}/links.
func (c *LinkController) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	var request createLinkRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	link, err := c.service.Create(r.Context(), services.CreateLinkInput{
		OrgID:  id.OrgID,
		UserID: id.UserID,
		URL:    request.URL,
		Slug:   request.Slug,
		Title:  request.Title,
		Tags:   request.Tags,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	c.purgeLink(r, link)
	w.Header().Set(httpx.HeaderLocation, "/orgs/"+link.OrgID+"/links/"+link.ID)
	httpx.WriteJSON(w, r, http.StatusCreated, c.linkResponse(r, link))
}

// List handles GET and HEAD /orgs/{orgID}/links.
func (c *LinkController) List(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	limit, offset, message := parsePagination(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}

	result, err := c.service.List(r.Context(), services.ListLinksInput{
		OrgID:  id.OrgID,
		Query:  r.URL.Query().Get("q"),
		Tags:   parseTags(r.URL.Query()),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}

	response := linkListResponse{
		Items:      make([]linkResponse, 0, len(result.Items)),
		NextOffset: result.NextOffset,
	}
	for _, link := range result.Items {
		response.Items = append(response.Items, c.linkResponse(r, link))
	}
	httpx.WriteJSON(w, r, http.StatusOK, response)
}

// Get handles GET and HEAD /orgs/{orgID}/links/{linkID}.
func (c *LinkController) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	link, err := c.service.Get(r.Context(), id.OrgID, chi.URLParam(r, "linkID"))
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, c.linkResponse(r, link))
}

// Update handles PATCH /orgs/{orgID}/links/{linkID}.
func (c *LinkController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	var request updateLinkRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	link, err := c.service.Update(r.Context(), services.UpdateLinkInput{
		OrgID:    id.OrgID,
		ID:       chi.URLParam(r, "linkID"),
		URL:      request.URL,
		Title:    request.Title,
		Tags:     request.Tags,
		Disabled: request.Disabled,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	c.purgeLink(r, link)
	httpx.WriteJSON(w, r, http.StatusOK, c.linkResponse(r, link))
}

// Delete handles DELETE /orgs/{orgID}/links/{linkID}.
func (c *LinkController) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}

	link, err := c.service.Delete(r.Context(), id.OrgID, chi.URLParam(r, "linkID"))
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	c.purgeLink(r, link)
	w.WriteHeader(http.StatusNoContent)
}

// Redirect handles the public GET /s/{slug} redirect.
func (c *LinkController) Redirect(w http.ResponseWriter, r *http.Request) {
	// Errors must never be cached. A successful redirect overrides this header
	// with the edge-cacheable policy below.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex")

	link, err := c.service.Resolve(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		if errors.Is(err, entities.ErrLinkNotFound) {
			httpx.WriteError(w, r, http.StatusNotFound, "link not found")
			return
		}
		logInternalError(c.logger, w, r, "link redirect failed", err)
		return
	}

	w.Header().Set("Cache-Control", shortLinkCacheControl)
	w.Header().Set(httpx.HeaderLocation, link.URL)
	w.WriteHeader(http.StatusFound)
}

func (c *LinkController) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *validation.Error
	switch {
	case errors.As(err, &validationErr):
		httpx.WriteError(w, r, http.StatusBadRequest, validationErr.Message)
	case errors.Is(err, entities.ErrLinkNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "link not found")
	case errors.Is(err, entities.ErrSlugTaken):
		httpx.WriteError(w, r, http.StatusConflict, "slug already in use")
	default:
		logInternalError(c.logger, w, r, "link request failed", err)
	}
}

type createLinkRequest struct {
	URL   string   `json:"url"`
	Slug  string   `json:"slug"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

type updateLinkRequest struct {
	URL      *string   `json:"url"`
	Title    *string   `json:"title"`
	Tags     *[]string `json:"tags"`
	Disabled *bool     `json:"disabled"`
}

type linkResponse struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"orgId"`
	UserID    string    `json:"userId"`
	Slug      string    `json:"slug"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Tags      []string  `json:"tags"`
	ShortPath string    `json:"shortPath"`
	ShortURL  string    `json:"shortUrl"`
	Disabled  bool      `json:"disabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type linkListResponse struct {
	Items      []linkResponse `json:"items"`
	NextOffset *int           `json:"nextOffset"`
}

const (
	shortLinkPathPrefix = "/s/"
	// Browsers revalidate immediately; shared caches such as Bunny may keep
	// the redirect for 10 minutes. Mutations purge the exact URL.
	shortLinkCacheControl = "public, max-age=0, s-maxage=600"
	purgeTimeout          = 2 * time.Second
)

func (c *LinkController) linkResponse(r *http.Request, link entities.Link) linkResponse {
	tags := link.Tags
	if tags == nil {
		tags = []string{}
	}

	shortPath := shortLinkPathPrefix + link.Slug
	baseURL := c.shortBaseURL
	if baseURL == "" {
		baseURL = requestOrigin(r)
	}

	return linkResponse{
		ID:        link.ID,
		OrgID:     link.OrgID,
		UserID:    link.UserID,
		Slug:      link.Slug,
		URL:       link.URL,
		Title:     link.Title,
		Tags:      tags,
		ShortPath: shortPath,
		ShortURL:  baseURL + shortPath,
		Disabled:  link.Disabled(),
		CreatedAt: link.CreatedAt,
		UpdatedAt: link.UpdatedAt,
	}
}

func (c *LinkController) purgeLink(r *http.Request, link entities.Link) {
	if c.purger == nil || c.shortBaseURL == "" || link.Slug == "" {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), purgeTimeout)
	defer cancel()

	target := c.shortBaseURL + shortLinkPathPrefix + link.Slug
	if err := c.purger.Purge(ctx, target); err != nil {
		c.logger.WarnContext(r.Context(), "purge short link cache",
			"slug", link.Slug,
			"error", err,
			"request_id", httpx.RequestID(r.Context()),
		)
	}
}

func requestOrigin(r *http.Request) string {
	scheme := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	if scheme != "http" && scheme != "https" {
		switch {
		case r.URL.Scheme == "http" || r.URL.Scheme == "https":
			scheme = r.URL.Scheme
		case r.TLS != nil:
			scheme = "https"
		default:
			scheme = "http"
		}
	}

	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = r.URL.Host
	}
	return scheme + "://" + host
}
