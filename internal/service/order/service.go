package order

import (
	"context"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

type Repository interface {
	PlaceOrder(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Place(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error) {
	if len(items) == 0 {
		return nil, model.ErrInvalid
	}
	for _, item := range items {
		if item.ProductID <= 0 || item.Quantity <= 0 {
			return nil, model.ErrInvalid
		}
	}

	return s.repo.PlaceOrder(ctx, userID, items)
}
