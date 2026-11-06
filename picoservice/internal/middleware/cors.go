package middleware

import (
	"net/http"
	"strings"
)

const (
	corsAllowedMethods = "GET, HEAD, POST, PATCH, DELETE, OPTIONS"
	corsAllowedHeaders = "Content-Type, Authorization"
	corsMaxAge         = "3600"
)

// CORS allows the configured origins with credentials and answers preflight
// requests. A "*" entry reflects any non-null request origin, which is the only
// way to combine allow-all with credentials. It is the single CORS layer for
// both the local server and Lambda; do not also configure Function URL CORS.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := false
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		switch origin {
		case "":
			continue
		case "*":
			allowAll = true
		default:
			allowed[strings.ToLower(origin)] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" || strings.EqualFold(origin, "null") {
				next.ServeHTTP(w, r)
				return
			}
			if !allowAll {
				if _, ok := allowed[strings.ToLower(origin)]; !ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
				w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
				w.Header().Set("Access-Control-Max-Age", corsMaxAge)
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
