package order

import (
	"context"
	"errors"
	"testing"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

type fakeRepo struct {
	placeOrderFn func(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error)
}

func (f *fakeRepo) PlaceOrder(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error) {
	return f.placeOrderFn(ctx, userID, items)
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
			name:    "некорректный product_id",
			items:   []model.OrderLineInput{{ProductID: -1, Quantity: 1}},
			wantErr: model.ErrInvalid,
		},
		{
			name:      "не хватает stock — ошибка репозитория пробрасывается как есть",
			items:     []model.OrderLineInput{{ProductID: 1, Quantity: 999}},
			repoErr:   model.ErrNoStock,
			wantErr:   model.ErrNoStock,
			callsRepo: true,
		},
		{
			name:      "hidden товар — ошибка репозитория пробрасывается как есть",
			items:     []model.OrderLineInput{{ProductID: 1, Quantity: 1}},
			repoErr:   model.ErrInvalid,
			wantErr:   model.ErrInvalid,
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
