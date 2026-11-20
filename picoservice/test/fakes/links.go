package fakes

import (
	"context"

	"picoservice/internal/entities"
	"picoservice/internal/services"
)

// LinkService is a configurable fake link service.
type LinkService struct {
	CreateFunc  func(context.Context, services.CreateLinkInput) (entities.Link, error)
	ListFunc    func(context.Context, services.ListLinksInput) (services.LinkList, error)
	GetFunc     func(context.Context, string, string) (entities.Link, error)
	ResolveFunc func(context.Context, string) (entities.Link, error)
	UpdateFunc  func(context.Context, services.UpdateLinkInput) (entities.Link, error)
	DeleteFunc  func(context.Context, string, string) (entities.Link, error)
}

// LinkPurger is a configurable fake CDN purger.
type LinkPurger struct {
	PurgeFunc func(context.Context, string) error
}

func (p *LinkPurger) Purge(ctx context.Context, url string) error {
	if p.PurgeFunc != nil {
		return p.PurgeFunc(ctx, url)
	}
	return nil
}

func (s *LinkService) Create(ctx context.Context, in services.CreateLinkInput) (entities.Link, error) {
	if s.CreateFunc != nil {
		return s.CreateFunc(ctx, in)
	}
	return entities.Link{}, nil
}

func (s *LinkService) List(ctx context.Context, in services.ListLinksInput) (services.LinkList, error) {
	if s.ListFunc != nil {
		return s.ListFunc(ctx, in)
	}
	return services.LinkList{Items: []entities.Link{}}, nil
}

func (s *LinkService) Get(ctx context.Context, orgID, id string) (entities.Link, error) {
	if s.GetFunc != nil {
		return s.GetFunc(ctx, orgID, id)
	}
	return entities.Link{}, nil
}

func (s *LinkService) Resolve(ctx context.Context, slug string) (entities.Link, error) {
	if s.ResolveFunc != nil {
		return s.ResolveFunc(ctx, slug)
	}
	return entities.Link{}, nil
}

func (s *LinkService) Update(ctx context.Context, in services.UpdateLinkInput) (entities.Link, error) {
	if s.UpdateFunc != nil {
		return s.UpdateFunc(ctx, in)
	}
	return entities.Link{}, nil
}

func (s *LinkService) Delete(ctx context.Context, orgID, id string) (entities.Link, error) {
	if s.DeleteFunc != nil {
		return s.DeleteFunc(ctx, orgID, id)
	}
	return entities.Link{}, nil
}
