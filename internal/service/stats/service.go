package stats

import (
	"context"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

type Repository interface {
	Get(ctx context.Context, sellerID int64) (*model.SalesStats, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, sellerID int64) (*model.SalesStats, error) {
	return s.repo.Get(ctx, sellerID)
}
