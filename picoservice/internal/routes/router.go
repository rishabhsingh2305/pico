// Package routes builds the application HTTP router, including middleware,
// domain subrouters, and JSON 404/405 responses.
package routes

import (
	"errors"
	"log/slog"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"picoservice/internal/controllers"
	"picoservice/internal/middleware"
)

// UserService is the user behavior required by the router.
type UserService interface {
	controllers.UserService
	middleware.UserLookup
}

// OrgService is the organization behavior required by the router.
type OrgService interface {
	controllers.OrgService
	middleware.OrgLookup
}

// MembershipService is the membership behavior required by the router.
type MembershipService interface {
	controllers.MembershipService
	middleware.MembershipLookup
}

// ProjectService is the project behavior required by the router.
type ProjectService interface {
	controllers.ProjectService
}

// ExpenseService is the expense behavior required by the router.
type ExpenseService interface {
	controllers.ExpenseService
}

// Deps contains the dependencies required to build the application router.
type Deps struct {
	Logger      *slog.Logger
	Pinger      controllers.Pinger
	Users       UserService
	Orgs        OrgService
	Memberships MembershipService
	Projects    ProjectService
	Expenses    ExpenseService
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
	orgController := controllers.NewOrgController(logger, deps.Orgs)
	membershipController := controllers.NewMembershipController(logger, deps.Memberships)
	projectController := controllers.NewProjectController(logger, deps.Projects)
	expenseController := controllers.NewExpenseController(logger, deps.Expenses)

	requireUser := middleware.RequireUser(logger, deps.Users)
	requireMember := middleware.RequireMember(logger, deps.Users, deps.Orgs, deps.Memberships)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.RequestIDHeader)
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.Recoverer(logger))

	healthRoutes(r, logger, deps.Pinger)
	r.Mount("/auth", authRoutes(userController))
	r.Mount("/users", userRoutes(userController, requireUser))
	r.Mount("/orgs", orgRoutes(orgController, membershipController, requireUser, requireMember))
	r.Mount("/projects", projectRoutes(projectController, requireMember))
	r.Mount("/expenses", expenseRoutes(expenseController, requireMember))

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
	case d.Orgs == nil:
		return errors.New("routes: org service is required")
	case d.Memberships == nil:
		return errors.New("routes: membership service is required")
	case d.Projects == nil:
		return errors.New("routes: project service is required")
	case d.Expenses == nil:
		return errors.New("routes: expense service is required")
	default:
		return nil
	}
}
