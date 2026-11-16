package fakes

import (
	"context"

	"picoservice/internal/entities"
	"picoservice/internal/services"
)

// CredentialService is a configurable fake share service.
type CredentialService struct {
	CreateFunc  func(context.Context, services.CreateCredentialInput) (services.CreateCredentialResult, error)
	ListFunc    func(context.Context, services.ListCredentialsInput) ([]entities.Credential, error)
	RevokeFunc  func(context.Context, string, string) error
	ResolveFunc func(context.Context, string) (entities.Credential, error)
}

func (s *CredentialService) Create(ctx context.Context, in services.CreateCredentialInput) (services.CreateCredentialResult, error) {
	if s.CreateFunc != nil {
		return s.CreateFunc(ctx, in)
	}
	return services.CreateCredentialResult{}, nil
}

func (s *CredentialService) List(ctx context.Context, in services.ListCredentialsInput) ([]entities.Credential, error) {
	if s.ListFunc != nil {
		return s.ListFunc(ctx, in)
	}
	return []entities.Credential{}, nil
}

func (s *CredentialService) Revoke(ctx context.Context, orgID, id string) error {
	if s.RevokeFunc != nil {
		return s.RevokeFunc(ctx, orgID, id)
	}
	return nil
}

func (s *CredentialService) ResolveToken(ctx context.Context, raw string) (entities.Credential, error) {
	if s.ResolveFunc != nil {
		return s.ResolveFunc(ctx, raw)
	}
	return entities.Credential{}, entities.ErrCredentialNotFound
}
