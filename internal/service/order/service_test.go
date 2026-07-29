package order

import (
	"context"
	"errors"
	"testing"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

type fakeRepo struct {
	placeOrderFn func(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error)
	listFn       func(ctx context.Context, f model.OrderFilter) (model.OrderListResult, error)
	getDetailFn  func(ctx context.Context, id int64) (*model.OrderDetail, error)
	gotFilter    model.OrderFilter
}

func (f *fakeRepo) PlaceOrder(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error) {
	return f.placeOrderFn(ctx, userID, items)
}

func (f *fakeRepo) List(ctx context.Context, filter model.OrderFilter) (model.OrderListResult, error) {
	f.gotFilter = filter
	if f.listFn == nil {
		return model.OrderListResult{}, nil
	}
	return f.listFn(ctx, filter)
}

func (f *fakeRepo) GetDetail(ctx context.Context, id int64) (*model.OrderDetail, error) {
	if f.getDetailFn == nil {
		return nil, model.ErrNotFound
	}
	return f.getDetailFn(ctx, id)
}

func TestService_Place(t *testing.T) {
	tests := []struct {
		name      string
		items     []model.OrderLineInput
		repoErr   error
		wantErr   error
		callsRepo bool
	}{
		{
			name:      "успешный заказ",
			items:     []model.OrderLineInput{{ProductID: 1, Quantity: 2}},
			callsRepo: true,
		},
		{
			name:    "пустой список позиций",
			items:   nil,
			wantErr: model.ErrInvalid,
		},
		{
			name:    "нулевое количество",
			items:   []model.OrderLineInput{{ProductID: 1, Quantity: 0}},
			wantErr: model.ErrInvalid,
		},
		{
			name:      "не хватает stock — пробрасывается как есть",
			items:     []model.OrderLineInput{{ProductID: 1, Quantity: 999}},
			repoErr:   model.ErrNoStock,
			wantErr:   model.ErrNoStock,
			callsRepo: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			repo := &fakeRepo{
				placeOrderFn: func(context.Context, int64, []model.OrderLineInput) (*model.Order, error) {
					called = true
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return &model.Order{ID: 1, Status: model.OrderStatusNew}, nil
				},
			}

			_, err := NewService(repo).Place(context.Background(), 7, tt.items)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if called != tt.callsRepo {
				t.Errorf("repo вызван = %v, want %v", called, tt.callsRepo)
			}
		})
	}
}

func TestService_List(t *testing.T) {
	repo := &fakeRepo{}

	_, err := NewService(repo).List(context.Background(), 7, model.OrderFilter{})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if repo.gotFilter.UserID == nil || *repo.gotFilter.UserID != 7 {
		t.Errorf("в репозиторий ушёл UserID=%v, want 7", repo.gotFilter.UserID)
	}
}

func TestService_GetDetail(t *testing.T) {
	tests := []struct {
		name        string
		requesterID int64
		order       *model.OrderDetail
		repoErr     error
		wantErr     error
	}{
		{
			name:        "свой заказ виден",
			requesterID: 7,
			order:       &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7}},
		},
		{
			name:        "чужой заказ — 403, а не 404",
			requesterID: 7,
			order:       &model.OrderDetail{Order: model.Order{ID: 1, UserID: 999}},
			wantErr:     model.ErrForbidden,
		},
		{
			name:        "заказа нет вообще",
			requesterID: 7,
			repoErr:     model.ErrNotFound,
			wantErr:     model.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{
				getDetailFn: func(context.Context, int64) (*model.OrderDetail, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return tt.order, nil
				},
			}

			_, err := NewService(repo).GetDetail(context.Background(), tt.requesterID, 1)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
		})
	}
}
