package routes

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"picoservice/internal/httpx"
)

// routeProbe is a flat copy of the route tree used to resolve Allow headers.
type routeProbe struct {
	router  *chi.Mux
	methods []string
}

// methodNotAllowed responds with a JSON 405 and derives the Allow header from
// a probe router built from the registered routes.
func methodNotAllowed(probe *routeProbe) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := make([]string, 0, len(probe.methods))
		for _, method := range probe.methods {
			if probe.router.Match(chi.NewRouteContext(), method, r.URL.Path) {
				allowed = append(allowed, method)
			}
		}

		if len(allowed) > 0 {
			w.Header().Set(httpx.HeaderAllow, strings.Join(allowed, ", "))
		}
		httpx.WriteError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// newMethodProbe builds a flat chi router from the application's route tree.
// The application router uses mounted subrouters, whose mount stubs match every
// method. The flat probe makes MethodNotAllowed's path matching reflect the
// actual methods registered for each path.
func newMethodProbe(routes chi.Routes) (*routeProbe, error) {
	probe := chi.NewRouter()
	seen := make(map[string]struct{})
	methodSet := make(map[string]struct{})
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	err := chi.Walk(routes, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = normalizeRoute(route)
		key := method + " " + route
		if _, ok := seen[key]; ok {
			return nil
		}
		seen[key] = struct{}{}
		methodSet[method] = struct{}{}

		probe.Method(method, route, handler)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk routes: %w", err)
	}

	methods := make([]string, 0, len(methodSet))
	for method := range methodSet {
		methods = append(methods, method)
	}
	sort.Slice(methods, func(i, j int) bool {
		ri, rj := methodRank(methods[i]), methodRank(methods[j])
		if ri != rj {
			return ri < rj
		}
		return methods[i] < methods[j]
	})

	return &routeProbe{router: probe, methods: methods}, nil
}

func normalizeRoute(route string) string {
	if route == "" {
		return "/"
	}
	if route != "/" {
		return strings.TrimSuffix(route, "/")
	}
	return route
}

func methodRank(method string) int {
	switch method {
	case http.MethodGet:
		return 0
	case http.MethodHead:
		return 1
	case http.MethodPost:
		return 2
	case http.MethodPut:
		return 3
	case http.MethodPatch:
		return 4
	case http.MethodDelete:
		return 5
	default:
		return 6
	}
}
