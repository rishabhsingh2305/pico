package controllers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/entities"
	"picoservice/internal/httpx"
	"picoservice/internal/services"
	"picoservice/internal/validation"
)

// AnalyticsSourceService is the analytics source behavior used by controllers.
type AnalyticsSourceService interface {
	Create(ctx context.Context, in services.CreateAnalyticsSourceInput) (entities.AnalyticsSource, error)
	List(ctx context.Context, in services.ListAnalyticsSourcesInput) (services.AnalyticsSourceList, error)
	Get(ctx context.Context, orgID, id string) (entities.AnalyticsSource, error)
	Update(ctx context.Context, in services.UpdateAnalyticsSourceInput) (entities.AnalyticsSource, error)
	Delete(ctx context.Context, orgID, id string) error
}

// AnalyticsStatsService is the capture stats behavior used by controllers.
type AnalyticsStatsService interface {
	Stats(ctx context.Context, in services.AnalyticsStatsInput) (services.AnalyticsStatsResult, error)
	BulkStats(ctx context.Context, in services.AnalyticsBulkStatsInput) (services.AnalyticsBulkStatsResult, error)
}

// AnalyticsController serves analytics source management and capture stats.
type AnalyticsController struct {
	logger       *slog.Logger
	sources      AnalyticsSourceService
	stats        AnalyticsStatsService
	shortBaseURL string
}

// NewAnalyticsController returns an analytics controller.
func NewAnalyticsController(
	logger *slog.Logger,
	sources AnalyticsSourceService,
	stats AnalyticsStatsService,
	shortBaseURL string,
) *AnalyticsController {
	if logger == nil {
		logger = slog.Default()
	}
	return &AnalyticsController{
		logger:       logger,
		sources:      sources,
		stats:        stats,
		shortBaseURL: strings.TrimRight(strings.TrimSpace(shortBaseURL), "/"),
	}
}

// Create handles POST /orgs/{orgID}/analytics/sources.
func (c *AnalyticsController) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	var request createAnalyticsSourceRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	source, err := c.sources.Create(r.Context(), services.CreateAnalyticsSourceInput{
		OrgID:       id.OrgID,
		Name:        request.Name,
		Description: request.Description,
		Events:      request.Events,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	w.Header().Set(httpx.HeaderLocation, "/orgs/"+source.OrgID+"/analytics/sources/"+source.ID)
	httpx.WriteJSON(w, r, http.StatusCreated, c.sourceResponse(r, source))
}

// List handles GET and HEAD /orgs/{orgID}/analytics/sources.
func (c *AnalyticsController) List(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	limit, offset, message := parsePagination(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > 64 {
		httpx.WriteError(w, r, http.StatusBadRequest, "q must be at most 64 characters")
		return
	}
	result, err := c.sources.List(r.Context(), services.ListAnalyticsSourcesInput{
		OrgID:  id.OrgID,
		Query:  query,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	response := analyticsSourceListResponse{
		Items:      make([]analyticsSourceResponse, 0, len(result.Items)),
		NextOffset: result.NextOffset,
	}
	for _, source := range result.Items {
		response.Items = append(response.Items, c.sourceResponse(r, source))
	}
	httpx.WriteJSON(w, r, http.StatusOK, response)
}

// Get handles GET and HEAD /orgs/{orgID}/analytics/sources/{sourceID}.
func (c *AnalyticsController) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	source, err := c.sources.Get(r.Context(), id.OrgID, chi.URLParam(r, "sourceID"))
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, c.sourceResponse(r, source))
}

// Update handles PATCH /orgs/{orgID}/analytics/sources/{sourceID}.
func (c *AnalyticsController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	var request updateAnalyticsSourceRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	source, err := c.sources.Update(r.Context(), services.UpdateAnalyticsSourceInput{
		OrgID:       id.OrgID,
		ID:          chi.URLParam(r, "sourceID"),
		Name:        request.Name,
		Description: request.Description,
		Events:      request.Events,
		Disabled:    request.Disabled,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, c.sourceResponse(r, source))
}

// Delete handles DELETE /orgs/{orgID}/analytics/sources/{sourceID}.
func (c *AnalyticsController) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	if err := c.sources.Delete(r.Context(), id.OrgID, chi.URLParam(r, "sourceID")); err != nil {
		c.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Stats handles GET and HEAD /orgs/{orgID}/analytics/sources/{sourceID}/stats.
func (c *AnalyticsController) Stats(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	sourceID := chi.URLParam(r, "sourceID")
	source, err := c.sources.Get(r.Context(), id.OrgID, sourceID)
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	from, to, message := analyticsRange(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}
	bot, message := analyticsBotFilter(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}
	result, err := c.stats.Stats(r.Context(), services.AnalyticsStatsInput{
		OrgID:     id.OrgID,
		SourceID:  source.ID,
		From:      from,
		To:        to,
		Interval:  r.URL.Query().Get("interval"),
		Event:     r.URL.Query().Get("event"),
		Countries: queryList(r, "country"),
		Devices:   queryList(r, "device"),
		Bot:       bot,
		Referrer:  r.URL.Query().Get("referrer"),
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, r, http.StatusOK, analyticsStatsResponseFrom(result))
}

// BulkStats handles GET and HEAD /orgs/{orgID}/analytics/stats.
func (c *AnalyticsController) BulkStats(w http.ResponseWriter, r *http.Request) {
	id, ok := requestIdentity(w, r)
	if !ok {
		return
	}
	ids, message := parseLinkIDs(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}
	from, to, message := analyticsRange(r)
	if message != "" {
		httpx.WriteError(w, r, http.StatusBadRequest, message)
		return
	}
	result, err := c.stats.BulkStats(r.Context(), services.AnalyticsBulkStatsInput{
		OrgID:     id.OrgID,
		SourceIDs: ids,
		From:      from,
		To:        to,
	})
	if err != nil {
		c.writeError(w, r, err)
		return
	}
	response := analyticsBulkStatsResponse{
		Items: make([]analyticsBulkStatsItem, 0, len(result.Items)),
	}
	for _, item := range result.Items {
		response.Items = append(response.Items, analyticsBulkStatsItem{
			SourceID: item.SourceID,
			Captures: item.Count,
		})
	}
	httpx.WriteJSON(w, r, http.StatusOK, response)
}

// AnalyticsPixel serves the public 1x1 capture pixel.
type AnalyticsPixel struct{}

// NewAnalyticsPixel returns a public capture pixel handler.
func NewAnalyticsPixel() *AnalyticsPixel {
	return &AnalyticsPixel{}
}

const transparentGIF = "GIF89a\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff!\xf9\x04\x01\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;"

// Pixel handles GET and HEAD /p/{key}.
func (p *AnalyticsPixel) Pixel(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Content-Length", strconv.Itoa(len(transparentGIF)))
	// Captures are counted from CDN request logs. Keep browsers from serving
	// the pixel out of local cache so views still reach the edge; the pull zone
	// may rewrite cache headers, but cached edge requests are still logged.
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Robots-Tag", "noindex")
	_, _ = w.Write([]byte(transparentGIF))
}

func (c *AnalyticsController) sourceResponse(r *http.Request, source entities.AnalyticsSource) analyticsSourceResponse {
	baseURL := c.shortBaseURL
	if baseURL == "" {
		baseURL = requestOrigin(r)
	}
	events := source.Events
	if events == nil {
		events = []string{}
	}
	pixelPath := "/p/" + source.Key
	return analyticsSourceResponse{
		ID:          source.ID,
		OrgID:       source.OrgID,
		Name:        source.Name,
		Description: source.Description,
		Key:         source.Key,
		Events:      events,
		PixelPath:   pixelPath,
		PixelURL:    baseURL + pixelPath,
		Disabled:    source.Disabled(),
		CreatedAt:   source.CreatedAt,
		UpdatedAt:   source.UpdatedAt,
	}
}

func (c *AnalyticsController) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *validation.Error
	switch {
	case errors.As(err, &validationErr):
		httpx.WriteError(w, r, http.StatusBadRequest, validationErr.Message)
	case errors.Is(err, entities.ErrAnalyticsSourceNotFound):
		httpx.WriteError(w, r, http.StatusNotFound, "analytics source not found")
	case errors.Is(err, entities.ErrAnalyticsKeyTaken):
		httpx.WriteError(w, r, http.StatusConflict, "analytics key already in use")
	default:
		logInternalError(c.logger, w, r, "analytics request failed", err)
	}
}

func analyticsRange(r *http.Request) (time.Time, time.Time, string) {
	from, message := parseTimeQuery(r, "from")
	if message != "" {
		return time.Time{}, time.Time{}, message
	}
	to, message := parseTimeQuery(r, "to")
	if message != "" {
		return time.Time{}, time.Time{}, message
	}
	now := time.Now().UTC()
	if from == nil {
		value := now.Add(-30 * 24 * time.Hour)
		from = &value
	}
	if to == nil {
		to = &now
	}
	return *from, *to, ""
}

func analyticsBotFilter(r *http.Request) (*bool, string) {
	raw := strings.TrimSpace(r.URL.Query().Get("bot"))
	if raw == "" {
		return nil, ""
	}
	switch strings.ToLower(raw) {
	case "true":
		value := true
		return &value, ""
	case "false":
		value := false
		return &value, ""
	default:
		return nil, "bot must be true or false"
	}
}

func queryList(r *http.Request, key string) []string {
	values := r.URL.Query()[key]
	items := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				items = append(items, part)
			}
		}
	}
	return items
}

type createAnalyticsSourceRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Events      []string `json:"events"`
}

type updateAnalyticsSourceRequest struct {
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	Events      *[]string `json:"events"`
	Disabled    *bool     `json:"disabled"`
}

type analyticsSourceResponse struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"orgId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Key         string    `json:"key"`
	Events      []string  `json:"events"`
	PixelPath   string    `json:"pixelPath"`
	PixelURL    string    `json:"pixelUrl"`
	Disabled    bool      `json:"disabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type analyticsSourceListResponse struct {
	Items      []analyticsSourceResponse `json:"items"`
	NextOffset *int                      `json:"nextOffset"`
}

type analyticsStatsResponse struct {
	Total      int64                       `json:"total"`
	Series     []analyticsSeriesResponse   `json:"series"`
	Events     []analyticsEventResponse    `json:"events"`
	Breakdowns analyticsBreakdownsResponse `json:"breakdowns"`
}

type analyticsSeriesResponse struct {
	Bucket time.Time `json:"bucket"`
	Count  int64     `json:"count"`
}

type analyticsEventResponse struct {
	Event string `json:"event"`
	Count int64  `json:"count"`
}

type analyticsDimensionResponse struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

type analyticsBreakdownsResponse struct {
	Countries []analyticsDimensionResponse `json:"countries"`
	Devices   []analyticsDimensionResponse `json:"devices"`
	Bots      []analyticsDimensionResponse `json:"bots"`
	Referrers []analyticsDimensionResponse `json:"referrers"`
}

type analyticsBulkStatsResponse struct {
	Items []analyticsBulkStatsItem `json:"items"`
}

type analyticsBulkStatsItem struct {
	SourceID string `json:"sourceId"`
	Captures int64  `json:"captures"`
}

func analyticsStatsResponseFrom(result services.AnalyticsStatsResult) analyticsStatsResponse {
	response := analyticsStatsResponse{
		Total:  result.Total,
		Series: make([]analyticsSeriesResponse, 0, len(result.Series)),
		Events: make([]analyticsEventResponse, 0, len(result.Events)),
		Breakdowns: analyticsBreakdownsResponse{
			Countries: make([]analyticsDimensionResponse, 0, len(result.Countries)),
			Devices:   make([]analyticsDimensionResponse, 0, len(result.Devices)),
			Bots:      make([]analyticsDimensionResponse, 0, len(result.Bots)),
			Referrers: make([]analyticsDimensionResponse, 0, len(result.Referrers)),
		},
	}
	for _, item := range result.Series {
		response.Series = append(response.Series, analyticsSeriesResponse{Bucket: item.Bucket, Count: item.Count})
	}
	for _, item := range result.Events {
		response.Events = append(response.Events, analyticsEventResponse{Event: item.Event, Count: item.Count})
	}
	for _, item := range result.Countries {
		response.Breakdowns.Countries = append(response.Breakdowns.Countries, analyticsDimensionResponse{Value: item.Value, Count: item.Count})
	}
	for _, item := range result.Devices {
		response.Breakdowns.Devices = append(response.Breakdowns.Devices, analyticsDimensionResponse{Value: item.Value, Count: item.Count})
	}
	for _, item := range result.Bots {
		response.Breakdowns.Bots = append(response.Breakdowns.Bots, analyticsDimensionResponse{Value: item.Value, Count: item.Count})
	}
	for _, item := range result.Referrers {
		response.Breakdowns.Referrers = append(response.Breakdowns.Referrers, analyticsDimensionResponse{Value: item.Value, Count: item.Count})
	}
	return response
}
