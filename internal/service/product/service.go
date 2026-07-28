package product

import (
	"context"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

type Repository interface {
	List(ctx context.Context, f model.ProductFilter) (model.ProductListResult, error)
	GetByID(ctx context.Context, id int64) (*model.Product, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, f model.ProductFilter) (model.ProductListResult, error) {
	if f.PriceMin != nil && f.PriceMax != nil && *f.PriceMin > *f.PriceMax {
		return model.ProductListResult{}, model.ErrInvalid
	}

	active := model.ProductStatusActive
	f.Status = &active

	return s.repo.List(ctx, f)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*model.Product, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Status != model.ProductStatusActive {
		return nil, model.ErrNotFound
	}
	return p, nil
}
