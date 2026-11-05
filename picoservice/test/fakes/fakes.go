// Package fakes provides test doubles for the router's service dependencies.
package fakes

import (
	"context"
	"log/slog"

	"picoservice/internal/controllers"
	"picoservice/internal/entities"
	"picoservice/internal/routes"
	"picoservice/internal/services"
)

// Services groups fake service implementations for router tests.
type Services struct {
	Users       *UserService
	Orgs        *OrgService
	Memberships *MembershipService
	Projects    *ProjectService
	Expenses    *ExpenseService
}

// NewServices returns fake services that accept any existing identity by
// default.
func NewServices() Services {
	return Services{
		Users:       &UserService{},
		Orgs:        &OrgService{},
		Memberships: &MembershipService{},
		Projects:    &ProjectService{},
		Expenses:    &ExpenseService{},
	}
}

// Deps returns router dependencies backed by the fake services.
func (s Services) Deps(logger *slog.Logger, pinger controllers.Pinger) routes.Deps {
	return routes.Deps{
		Logger:      logger,
		Pinger:      pinger,
		Users:       s.Users,
		Orgs:        s.Orgs,
		Memberships: s.Memberships,
		Projects:    s.Projects,
		Expenses:    s.Expenses,
	}
}

// UserService is a configurable fake user service.
type UserService struct {
	CreateFunc             func(context.Context, services.CreateUserInput) (entities.User, error)
	ListFunc               func(context.Context, services.ListUsersInput) ([]entities.User, error)
	GetFunc                func(context.Context, string) (entities.User, error)
	UpdateFunc             func(context.Context, services.UpdateUserInput) (entities.User, error)
	ChangePasswordFunc     func(context.Context, services.ChangePasswordInput) error
	VerifyEmailFunc        func(context.Context, string) (entities.User, error)
	ResendVerificationFunc func(context.Context, string) error
	ExistsFunc             func(context.Context, string) (bool, error)
	Existing               map[string]bool
}

func (s *UserService) Create(ctx context.Context, in services.CreateUserInput) (entities.User, error) {
	if s.CreateFunc != nil {
		return s.CreateFunc(ctx, in)
	}
	return entities.User{}, nil
}

func (s *UserService) List(ctx context.Context, in services.ListUsersInput) ([]entities.User, error) {
	if s.ListFunc != nil {
		return s.ListFunc(ctx, in)
	}
	return []entities.User{}, nil
}

func (s *UserService) Get(ctx context.Context, id string) (entities.User, error) {
	if s.GetFunc != nil {
		return s.GetFunc(ctx, id)
	}
	return entities.User{}, nil
}

func (s *UserService) Update(ctx context.Context, in services.UpdateUserInput) (entities.User, error) {
	if s.UpdateFunc != nil {
		return s.UpdateFunc(ctx, in)
	}
	return entities.User{}, nil
}

func (s *UserService) ChangePassword(ctx context.Context, in services.ChangePasswordInput) error {
	if s.ChangePasswordFunc != nil {
		return s.ChangePasswordFunc(ctx, in)
	}
	return nil
}

func (s *UserService) VerifyEmail(ctx context.Context, token string) (entities.User, error) {
	if s.VerifyEmailFunc != nil {
		return s.VerifyEmailFunc(ctx, token)
	}
	return entities.User{}, nil
}

func (s *UserService) ResendVerification(ctx context.Context, email string) error {
	if s.ResendVerificationFunc != nil {
		return s.ResendVerificationFunc(ctx, email)
	}
	return nil
}

func (s *UserService) Exists(ctx context.Context, id string) (bool, error) {
	if s.ExistsFunc != nil {
		return s.ExistsFunc(ctx, id)
	}
	if s.Existing != nil {
		return s.Existing[id], nil
	}
	return true, nil
}

// OrgService is a configurable fake organization service.
type OrgService struct {
	CreateFunc func(context.Context, services.CreateOrgInput) (entities.Org, error)
	ListFunc   func(context.Context, services.ListOrgsInput) ([]entities.Org, error)
	GetFunc    func(context.Context, string) (entities.Org, error)
	UpdateFunc func(context.Context, services.UpdateOrgInput) (entities.Org, error)
	ExistsFunc func(context.Context, string) (bool, error)
	Existing   map[string]bool
}

func (s *OrgService) Create(ctx context.Context, in services.CreateOrgInput) (entities.Org, error) {
	if s.CreateFunc != nil {
		return s.CreateFunc(ctx, in)
	}
	return entities.Org{}, nil
}

func (s *OrgService) List(ctx context.Context, in services.ListOrgsInput) ([]entities.Org, error) {
	if s.ListFunc != nil {
		return s.ListFunc(ctx, in)
	}
	return []entities.Org{}, nil
}

func (s *OrgService) Get(ctx context.Context, id string) (entities.Org, error) {
	if s.GetFunc != nil {
		return s.GetFunc(ctx, id)
	}
	return entities.Org{}, nil
}

func (s *OrgService) Update(ctx context.Context, in services.UpdateOrgInput) (entities.Org, error) {
	if s.UpdateFunc != nil {
		return s.UpdateFunc(ctx, in)
	}
	return entities.Org{}, nil
}

func (s *OrgService) Exists(ctx context.Context, id string) (bool, error) {
	if s.ExistsFunc != nil {
		return s.ExistsFunc(ctx, id)
	}
	if s.Existing != nil {
		return s.Existing[id], nil
	}
	return true, nil
}

// MembershipService is a configurable fake membership service.
type MembershipService struct {
	AddFunc    func(context.Context, services.CreateMembershipInput) (entities.Membership, error)
	ListFunc   func(context.Context, services.ListMembershipsInput) ([]entities.Membership, error)
	RemoveFunc func(context.Context, string, string, string) error
	ExistsFunc func(context.Context, string, string) (bool, error)
	Existing   map[string]bool
}

func (s *MembershipService) Add(ctx context.Context, in services.CreateMembershipInput) (entities.Membership, error) {
	if s.AddFunc != nil {
		return s.AddFunc(ctx, in)
	}
	return entities.Membership{}, nil
}

func (s *MembershipService) List(ctx context.Context, in services.ListMembershipsInput) ([]entities.Membership, error) {
	if s.ListFunc != nil {
		return s.ListFunc(ctx, in)
	}
	return []entities.Membership{}, nil
}

func (s *MembershipService) Remove(ctx context.Context, actorID, orgID, userID string) error {
	if s.RemoveFunc != nil {
		return s.RemoveFunc(ctx, actorID, orgID, userID)
	}
	return nil
}

func (s *MembershipService) Exists(ctx context.Context, orgID, userID string) (bool, error) {
	if s.ExistsFunc != nil {
		return s.ExistsFunc(ctx, orgID, userID)
	}
	if s.Existing != nil {
		return s.Existing[orgID+"/"+userID], nil
	}
	return true, nil
}

// ProjectService is a configurable fake project service.
type ProjectService struct {
	CreateFunc func(context.Context, services.CreateProjectInput) (entities.Project, error)
	ListFunc   func(context.Context, services.ListProjectsInput) ([]entities.Project, error)
	GetFunc    func(context.Context, string, string) (entities.Project, error)
	UpdateFunc func(context.Context, services.UpdateProjectInput) (entities.Project, error)
}

func (s *ProjectService) Create(ctx context.Context, in services.CreateProjectInput) (entities.Project, error) {
	if s.CreateFunc != nil {
		return s.CreateFunc(ctx, in)
	}
	return entities.Project{}, nil
}

func (s *ProjectService) List(ctx context.Context, in services.ListProjectsInput) ([]entities.Project, error) {
	if s.ListFunc != nil {
		return s.ListFunc(ctx, in)
	}
	return []entities.Project{}, nil
}

func (s *ProjectService) Get(ctx context.Context, orgID, id string) (entities.Project, error) {
	if s.GetFunc != nil {
		return s.GetFunc(ctx, orgID, id)
	}
	return entities.Project{}, nil
}

func (s *ProjectService) Update(ctx context.Context, in services.UpdateProjectInput) (entities.Project, error) {
	if s.UpdateFunc != nil {
		return s.UpdateFunc(ctx, in)
	}
	return entities.Project{}, nil
}

// ExpenseService is a configurable fake expense service.
type ExpenseService struct {
	CreateFunc func(context.Context, services.CreateExpenseInput) (entities.Expense, error)
	ListFunc   func(context.Context, services.ListExpensesInput) (services.ExpenseList, error)
	GetFunc    func(context.Context, string, string) (entities.Expense, error)
	UpdateFunc func(context.Context, services.UpdateExpenseInput) (entities.Expense, error)
	DeleteFunc func(context.Context, string, string) error
}

func (s *ExpenseService) Create(ctx context.Context, in services.CreateExpenseInput) (entities.Expense, error) {
	if s.CreateFunc != nil {
		return s.CreateFunc(ctx, in)
	}
	return entities.Expense{}, nil
}

func (s *ExpenseService) List(ctx context.Context, in services.ListExpensesInput) (services.ExpenseList, error) {
	if s.ListFunc != nil {
		return s.ListFunc(ctx, in)
	}
	return services.ExpenseList{Items: []entities.Expense{}}, nil
}

func (s *ExpenseService) Get(ctx context.Context, orgID, id string) (entities.Expense, error) {
	if s.GetFunc != nil {
		return s.GetFunc(ctx, orgID, id)
	}
	return entities.Expense{}, nil
}

func (s *ExpenseService) Update(ctx context.Context, in services.UpdateExpenseInput) (entities.Expense, error) {
	if s.UpdateFunc != nil {
		return s.UpdateFunc(ctx, in)
	}
	return entities.Expense{}, nil
}

func (s *ExpenseService) Delete(ctx context.Context, orgID, id string) error {
	if s.DeleteFunc != nil {
		return s.DeleteFunc(ctx, orgID, id)
	}
	return nil
}
