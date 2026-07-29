package order

import (
	"context"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

type Repository interface {
	PlaceOrder(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error)
	List(ctx context.Context, f model.OrderFilter) (model.OrderListResult, error)
	GetDetail(ctx context.Context, id int64) (*model.OrderDetail, error)
	ListSellerOrders(ctx context.Context, sellerID int64, f model.OrderFilter) (model.OrderListResult, error)
	GetSellerOrder(ctx context.Context, sellerID int64, orderID int64) (*model.OrderDetail, error)
	Pay(ctx context.Context, id int64) (*model.Order, error)
	Cancel(ctx context.Context, id int64) (*model.Order, error)
	Ship(ctx context.Context, id int64) (*model.Order, error)
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

func (s *Service) List(ctx context.Context, userID int64, f model.OrderFilter) (model.OrderListResult, error) {
	f.UserID = &userID
	return s.repo.List(ctx, f)
}

func (s *Service) GetDetail(ctx context.Context, userID, orderID int64) (*model.OrderDetail, error) {
	detail, err := s.repo.GetDetail(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if detail.UserID != userID {
		return nil, model.ErrForbidden
	}
	return detail, nil
}

func (s *Service) ListSellerOrders(ctx context.Context, sellerID int64, f model.OrderFilter) (model.OrderListResult, error) {
	return s.repo.ListSellerOrders(ctx, sellerID, f)
}

func (s *Service) GetSellerOrder(ctx context.Context, sellerID int64, orderID int64) (*model.OrderDetail, error) {
	return s.repo.GetSellerOrder(ctx, sellerID, orderID)
}

func (s *Service) Pay(ctx context.Context, userID, orderID int64) (*model.Order, error) {
	detail, err := s.repo.GetDetail(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if detail.UserID != userID {
		return nil, model.ErrForbidden
	}
	return s.repo.Pay(ctx, orderID)
}

func (s *Service) Cancel(ctx context.Context, userID, orderID int64) (*model.Order, error) {
	detail, err := s.repo.GetDetail(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if detail.UserID != userID {
		return nil, model.ErrForbidden
	}
	return s.repo.Cancel(ctx, orderID)
}

func (s *Service) Ship(ctx context.Context, sellerID, orderID int64) (*model.Order, error) {
	if _, err := s.repo.GetSellerOrder(ctx, sellerID, orderID); err != nil {
		return nil, err
	}
	return s.repo.Ship(ctx, orderID)
}
