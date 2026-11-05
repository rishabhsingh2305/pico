package controllers

import (
	"net/http"
	"strconv"
)

func parsePagination(r *http.Request) (limit, offset int, message string) {
	query := r.URL.Query()

	if raw := query.Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, "limit must be an integer"
		}
		limit = value
	}

	if raw := query.Get("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, "offset must be an integer"
		}
		offset = value
	}

	return limit, offset, ""
}
