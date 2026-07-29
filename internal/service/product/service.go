package product

import (
	"context"
	"strings"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

type Repository interface {
	List(ctx context.Context, f model.ProductFilter) (model.ProductListResult, error)
	GetByID(ctx context.Context, id int64) (*model.Product, error)
	Create(ctx context.Context, p model.Product) (model.Product, error)
	Update(ctx context.Context, p model.Product) (model.Product, error)
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

func (s *Service) Create(ctx context.Context, sellerID int64, in model.CreateProductInput) (model.Product, error) {
	if strings.TrimSpace(in.Name) == "" {
		return model.Product{}, model.ErrInvalid
	}
	if in.Price <= 0 {
		return model.Product{}, model.ErrInvalid
	}
	if in.Stock < 0 {
		return model.Product{}, model.ErrInvalid
	}

	p := model.Product{
		SellerID: sellerID,
		Name:     in.Name,
		Category: in.Category,
		Price:    in.Price,
		Stock:    in.Stock,
		Status:   model.ProductStatusActive,
	}

	return s.repo.Create(ctx, p)
}

func (s *Service) getOwned(ctx context.Context, sellerID, id int64) (*model.Product, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.SellerID != sellerID {
		return nil, model.ErrForbidden
	}
	return p, nil
}

func (s *Service) Update(ctx context.Context, sellerID, id int64, in model.UpdateProductInput) (model.Product, error) {
	p, err := s.getOwned(ctx, sellerID, id)
	if err != nil {
		return model.Product{}, err
	}

	if in.Name != nil {
		if strings.TrimSpace(*in.Name) == "" {
			return model.Product{}, model.ErrInvalid
		}
		p.Name = *in.Name
	}
	if in.Category != nil {
		p.Category = *in.Category
	}
	if in.Price != nil {
		if *in.Price <= 0 {
			return model.Product{}, model.ErrInvalid
		}
		p.Price = *in.Price
	}
	if in.Stock != nil {
		if *in.Stock < 0 {
			return model.Product{}, model.ErrInvalid
		}
		p.Stock = *in.Stock
	}
	if in.Status != nil {
		if *in.Status != model.ProductStatusActive && *in.Status != model.ProductStatusHidden {
			return model.Product{}, model.ErrInvalid
		}
		p.Status = *in.Status
	}

	return s.repo.Update(ctx, *p)
}

func (s *Service) Delete(ctx context.Context, sellerID, id int64) error {
	p, err := s.getOwned(ctx, sellerID, id)
	if err != nil {
		return err
	}
	p.Status = model.ProductStatusHidden
	_, err = s.repo.Update(ctx, *p)
	return err
}

func (s *Service) UpdatePhoto(ctx context.Context, sellerID, id int64, path string) (model.Product, error) {
	p, err := s.getOwned(ctx, sellerID, id)
	if err != nil {
		return model.Product{}, err
	}
	p.PhotoPath = &path
	return s.repo.Update(ctx, *p)
}
