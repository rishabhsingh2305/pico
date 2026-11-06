// Package app wires the application's DAO implementations, services, and routes
// for the Lambda and local server entrypoints.
package app

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"picoservice/internal/dao/mongodb"
	"picoservice/internal/email"
	"picoservice/internal/routes"
	"picoservice/internal/services"
)

// New builds the application handler from the configured MongoDB client and
// authentication settings.
func New(logger *slog.Logger, db *mongodb.Client, cfg Config) (http.Handler, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if db == nil {
		return nil, errors.New("app: mongodb client is required")
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("app: invalid config: %w", err)
	}

	userStore := mongodb.NewUserStore(db)
	orgStore := mongodb.NewOrgStore(db)
	membershipStore := mongodb.NewMembershipStore(db)
	projectStore := mongodb.NewProjectStore(db)
	expenseStore := mongodb.NewExpenseStore(db)

	tokens, err := services.NewVerificationTokens(cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("app: create verification tokens: %w", err)
	}
	mailer, err := email.NewSMTP(email.Config{
		Addr:     cfg.SMTPAddr,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.EmailFrom,
	})
	if err != nil {
		return nil, fmt.Errorf("app: create mailer: %w", err)
	}
	verification := services.NewEmailVerificationService(userStore, tokens, mailer, cfg.PublicBaseURL)

	hasher := services.NewBcryptHasher()
	userService := services.NewUserService(userStore, hasher, verification)
	authService, err := services.NewAuthService(userStore, hasher, cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("app: create auth service: %w", err)
	}
	orgService := services.NewOrgService(orgStore, membershipStore)
	membershipService := services.NewMembershipService(membershipStore, orgStore, userStore)
	projectService := services.NewProjectService(projectStore)
	expenseService := services.NewExpenseService(expenseStore, projectStore)

	return routes.New(routes.Deps{
		Logger:         logger,
		Pinger:         db,
		Users:          userService,
		Auth:           authService,
		Orgs:           orgService,
		Memberships:    membershipService,
		Projects:       projectService,
		Expenses:       expenseService,
		SecureCookies:  strings.HasPrefix(cfg.PublicBaseURL, "https://"),
		AllowedOrigins: cfg.AllowedOrigins(),
	})
}
