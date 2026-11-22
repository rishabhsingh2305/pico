// Package routes builds the application HTTP router, including middleware,
// domain subrouters, and JSON 404/405 responses.
package routes

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"picoservice/internal/authz"
	"picoservice/internal/controllers"
	"picoservice/internal/middleware"
)

// UserService is the user behavior required by the router.
type UserService interface {
	controllers.UserService
}

// AuthService is the authentication behavior required by the router.
type AuthService interface {
	controllers.AuthService
	middleware.AccessTokens
}

// OrgService is the organization behavior required by the router.
type OrgService interface {
	controllers.OrgService
}

// MembershipService is the membership behavior required by the router.
type MembershipService interface {
	controllers.MembershipService
}

// CredentialService is the share behavior required by the router.
type CredentialService interface {
	controllers.CredentialService
	middleware.CredentialResolver
}

// ProjectService is the project behavior required by the router.
type ProjectService interface {
	controllers.ProjectService
}

// ExpenseService is the expense behavior required by the router.
type ExpenseService interface {
	controllers.ExpenseService
}

// LinkService is the short-link behavior required by the router.
type LinkService interface {
	controllers.LinkService
}

// Deps contains the dependencies required to build the application router.
type Deps struct {
	Logger           *slog.Logger
	Pinger           controllers.Pinger
	Users            UserService
	Auth             AuthService
	Orgs             OrgService
	Memberships      MembershipService
	Projects         ProjectService
	Expenses         ExpenseService
	Links            LinkService
	LinkAnalytics    controllers.LinkStatsService
	AnalyticsSources controllers.AnalyticsSourceService
	AnalyticsStats   controllers.AnalyticsStatsService
	LinkPurger       controllers.LinkPurger
	ShortBaseURL     string
	Credentials      CredentialService
	Policy           *authz.Policy
	SecureCookies    bool
	AllowedOrigins   []string
}

// New builds the application router.
func New(deps Deps) (*chi.Mux, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}

	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	userController := controllers.NewUserController(logger, deps.Users)
	authController := controllers.NewAuthController(logger, deps.Auth, deps.SecureCookies)
	orgController := controllers.NewOrgController(logger, deps.Orgs)
	membershipController := controllers.NewMembershipController(logger, deps.Memberships)
	projectController := controllers.NewProjectController(logger, deps.Projects)
	expenseController := controllers.NewExpenseController(logger, deps.Expenses)
	linkController := controllers.NewLinkController(logger, deps.Links, deps.ShortBaseURL, deps.LinkPurger)
	linkAnalyticsController := controllers.NewLinkAnalyticsController(logger, deps.LinkAnalytics)
	analyticsController := controllers.NewAnalyticsController(
		logger,
		deps.AnalyticsSources,
		deps.AnalyticsStats,
		deps.ShortBaseURL,
	)
	analyticsPixel := controllers.NewAnalyticsPixel()
	credentialController := controllers.NewCredentialController(logger, deps.Credentials)

	requireAuth := middleware.RequireAuth(deps.Auth)
	requireOrg := func(next http.Handler) http.Handler {
		return middleware.Authenticate(logger, deps.Auth, deps.Credentials)(
			middleware.RequireOrgAccess(logger, deps.Policy)(next),
		)
	}

	r := chi.NewRouter()
	r.Use(middleware.CORS(deps.AllowedOrigins))
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.RequestIDHeader)
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.Recoverer(logger))

	healthRoutes(r, logger, deps.Pinger)
	r.Mount("/auth", authRoutes(userController, authController))
	r.Mount("/users", userRoutes(userController, requireAuth))
	r.Mount("/orgs", orgRoutes(
		orgController,
		membershipController,
		credentialController,
		linkController,
		linkAnalyticsController,
		analyticsController,
		requireAuth,
		requireOrg,
	))
	r.Mount("/projects", projectRoutes(projectController, requireOrg))
	r.Mount("/expenses", expenseRoutes(expenseController, requireOrg))
	r.Mount("/s", redirectRoutes(linkController))
	r.Mount("/p", pixelRoutes(analyticsPixel))

	probe, err := newMethodProbe(r)
	if err != nil {
		return nil, err
	}

	r.NotFound(controllers.NotFound())
	r.MethodNotAllowed(methodNotAllowed(probe))

	return r, nil
}

func (d Deps) validate() error {
	switch {
	case d.Pinger == nil:
		return errors.New("routes: pinger is required")
	case d.Users == nil:
		return errors.New("routes: user service is required")
	case d.Auth == nil:
		return errors.New("routes: auth service is required")
	case d.AllowedOrigins == nil || len(d.AllowedOrigins) == 0:
		return errors.New("routes: allowed origins are required")
	case d.Orgs == nil:
		return errors.New("routes: org service is required")
	case d.Memberships == nil:
		return errors.New("routes: membership service is required")
	case d.Projects == nil:
		return errors.New("routes: project service is required")
	case d.Expenses == nil:
		return errors.New("routes: expense service is required")
	case d.Links == nil:
		return errors.New("routes: link service is required")
	case d.LinkAnalytics == nil:
		return errors.New("routes: link analytics service is required")
	case d.AnalyticsSources == nil:
		return errors.New("routes: analytics source service is required")
	case d.AnalyticsStats == nil:
		return errors.New("routes: analytics stats service is required")
	case (d.ShortBaseURL == "") != (d.LinkPurger == nil):
		return errors.New("routes: short base URL and link purger must be set together")
	case d.Credentials == nil:
		return errors.New("routes: credential service is required")
	case d.Policy == nil:
		return errors.New("routes: authz policy is required")
	default:
		return nil
	}
}
