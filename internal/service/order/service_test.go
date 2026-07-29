package order

import (
	"context"
	"errors"
	"testing"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

type fakeRepo struct {
	placeOrderFn       func(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error)
	listFn             func(ctx context.Context, f model.OrderFilter) (model.OrderListResult, error)
	getDetailFn        func(ctx context.Context, id int64) (*model.OrderDetail, error)
	listSellerOrdersFn func(ctx context.Context, sellerID int64, f model.OrderFilter) (model.OrderListResult, error)
	getSellerOrderFn   func(ctx context.Context, sellerID int64, orderID int64) (*model.OrderDetail, error)
	gotFilter          model.OrderFilter
	payFn              func(ctx context.Context, id int64) (*model.Order, error)
	cancelFn           func(ctx context.Context, id int64) (*model.Order, error)
	shipFn             func(ctx context.Context, id int64) (*model.Order, error)
	payCalled          bool
	cancelCalled       bool
	shipCalled         bool
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

func (f *fakeRepo) ListSellerOrders(
	ctx context.Context,
	sellerID int64,
	filter model.OrderFilter,
) (model.OrderListResult, error) {

	if f.listSellerOrdersFn == nil {
		return model.OrderListResult{}, nil
	}

	return f.listSellerOrdersFn(ctx, sellerID, filter)
}

func (f *fakeRepo) GetSellerOrder(ctx context.Context, sellerID int64, orderID int64) (*model.OrderDetail, error) {
	if f.getSellerOrderFn == nil {
		return nil, model.ErrNotFound
	}
	return f.getSellerOrderFn(ctx, sellerID, orderID)
}

func (f *fakeRepo) Ship(ctx context.Context, id int64) (*model.Order, error) {
	f.shipCalled = true
	return f.shipFn(ctx, id)
}

func (f *fakeRepo) Pay(ctx context.Context, id int64) (*model.Order, error) {
	f.payCalled = true
	return f.payFn(ctx, id)
}

func (f *fakeRepo) Cancel(ctx context.Context, id int64) (*model.Order, error) {
	f.cancelCalled = true
	return f.cancelFn(ctx, id)
}

func TestService_Place(t *testing.T) {
	tests := []struct {
		name      string
		items     []model.OrderLineInput
		repoErr   error
		wantErr   error
		callsRepo bool
	}{
		{name: "успешный заказ", items: []model.OrderLineInput{{ProductID: 1, Quantity: 2}}, callsRepo: true},
		{name: "пустой список позиций", items: nil, wantErr: model.ErrInvalid},
		{name: "нулевое количество", items: []model.OrderLineInput{{ProductID: 1, Quantity: 0}}, wantErr: model.ErrInvalid},
		{
			name: "не хватает stock — пробрасывается как есть", items: []model.OrderLineInput{{ProductID: 1, Quantity: 999}},
			repoErr: model.ErrNoStock, wantErr: model.ErrNoStock, callsRepo: true,
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
		{name: "свой заказ виден", requesterID: 7, order: &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7}}},
		{
			name: "чужой заказ — 403, а не 404", requesterID: 7,
			order: &model.OrderDetail{Order: model.Order{ID: 1, UserID: 999}}, wantErr: model.ErrForbidden,
		},
		{name: "заказа нет вообще", requesterID: 7, repoErr: model.ErrNotFound, wantErr: model.ErrNotFound},
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

func TestService_Pay(t *testing.T) {
	tests := []struct {
		name        string
		requesterID int64
		order       *model.OrderDetail
		detailErr   error
		payErr      error
		wantErr     error
		callsPay    bool
	}{
		{
			name: "успешная оплата своего заказа", requesterID: 7,
			order:    &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7, Status: model.OrderStatusNew}},
			callsPay: true,
		},
		{
			name: "чужой заказ — 403, Pay не вызывается", requesterID: 7,
			order:   &model.OrderDetail{Order: model.Order{ID: 1, UserID: 999, Status: model.OrderStatusNew}},
			wantErr: model.ErrForbidden, callsPay: false,
		},
		{
			name: "заказа нет — 404, Pay не вызывается", requesterID: 7,
			detailErr: model.ErrNotFound, wantErr: model.ErrNotFound, callsPay: false,
		},
		{
			name: "уже оплачен — 400 из репозитория", requesterID: 7,
			order:  &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7, Status: model.OrderStatusPaid}},
			payErr: model.ErrInvalid, wantErr: model.ErrInvalid, callsPay: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{
				getDetailFn: func(context.Context, int64) (*model.OrderDetail, error) {
					if tt.detailErr != nil {
						return nil, tt.detailErr
					}
					return tt.order, nil
				},
				payFn: func(context.Context, int64) (*model.Order, error) {
					if tt.payErr != nil {
						return nil, tt.payErr
					}
					return &model.Order{ID: 1, Status: model.OrderStatusPaid}, nil
				},
			}

			_, err := NewService(repo).Pay(context.Background(), tt.requesterID, 1)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if repo.payCalled != tt.callsPay {
				t.Errorf("Pay вызван = %v, want %v", repo.payCalled, tt.callsPay)
			}
		})
	}
}

func TestService_Cancel(t *testing.T) {
	tests := []struct {
		name        string
		requesterID int64
		order       *model.OrderDetail
		cancelErr   error
		wantErr     error
		callsCancel bool
	}{
		{
			name: "успешная отмена своего заказа", requesterID: 7,
			order:       &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7, Status: model.OrderStatusNew}},
			callsCancel: true,
		},
		{
			name: "чужой заказ — 403, Cancel не вызывается", requesterID: 7,
			order:   &model.OrderDetail{Order: model.Order{ID: 1, UserID: 999, Status: model.OrderStatusNew}},
			wantErr: model.ErrForbidden, callsCancel: false,
		},
		{
			name: "уже оплачен — отменить нельзя, 400 из репозитория", requesterID: 7,
			order:     &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7, Status: model.OrderStatusPaid}},
			cancelErr: model.ErrInvalid, wantErr: model.ErrInvalid, callsCancel: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{
				getDetailFn: func(context.Context, int64) (*model.OrderDetail, error) {
					return tt.order, nil
				},
				cancelFn: func(context.Context, int64) (*model.Order, error) {
					if tt.cancelErr != nil {
						return nil, tt.cancelErr
					}
					return &model.Order{ID: 1, Status: model.OrderStatusCancelled}, nil
				},
			}

			_, err := NewService(repo).Cancel(context.Background(), tt.requesterID, 1)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if repo.cancelCalled != tt.callsCancel {
				t.Errorf("Cancel вызван = %v, want %v", repo.cancelCalled, tt.callsCancel)
			}
		})
	}
}

func TestService_Ship(t *testing.T) {
	tests := []struct {
		name           string
		sellerID       int64
		order          *model.OrderDetail
		sellerOrderErr error
		shipErr        error
		wantErr        error
		callsShip      bool
	}{
		{
			name:      "успешная отгрузка своим товаром в заказе",
			sellerID:  3,
			order:     &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7, Status: model.OrderStatusPaid}},
			callsShip: true,
		},
		{
			name:           "заказ без товаров этого продавца — 404, Ship не вызывается",
			sellerID:       999,
			sellerOrderErr: model.ErrNotFound,
			wantErr:        model.ErrNotFound,
			callsShip:      false,
		},
		{
			name:      "заказ не оплачен — 400 из репозитория",
			sellerID:  3,
			order:     &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7, Status: model.OrderStatusNew}},
			shipErr:   model.ErrInvalid,
			wantErr:   model.ErrInvalid,
			callsShip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{
				getSellerOrderFn: func(context.Context, int64, int64) (*model.OrderDetail, error) {
					if tt.sellerOrderErr != nil {
						return nil, tt.sellerOrderErr
					}
					return tt.order, nil
				},
				shipFn: func(context.Context, int64) (*model.Order, error) {
					if tt.shipErr != nil {
						return nil, tt.shipErr
					}
					return &model.Order{ID: 1, Status: model.OrderStatusShipped}, nil
				},
			}

			_, err := NewService(repo).Ship(context.Background(), tt.sellerID, 1)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if repo.shipCalled != tt.callsShip {
				t.Errorf("Ship вызван = %v, want %v", repo.shipCalled, tt.callsShip)
			}
		})
	}
}
