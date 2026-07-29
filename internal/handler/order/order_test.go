package order

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	orderservice "github.com/KhikmatovaNozee/orderFlow/internal/service/order"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeRepo struct {
	placeOrderFn       func(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error)
	listFn             func(ctx context.Context, f model.OrderFilter) (model.OrderListResult, error)
	getDetailFn        func(ctx context.Context, id int64) (*model.OrderDetail, error)
	listSellerOrdersFn func(ctx context.Context, sellerID int64, f model.OrderFilter) (model.OrderListResult, error)
	getByIDFn          func(ctx context.Context, id int64) (*model.Order, error)
	updateStatusFn     func(ctx context.Context, id int64, status string) error
	getSellerOrderFn   func(ctx context.Context, sellerID int64, orderID int64) (*model.OrderDetail, error)
}

func (f *fakeRepo) PlaceOrder(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error) {
	return f.placeOrderFn(ctx, userID, items)
}

func (f *fakeRepo) List(ctx context.Context, filter model.OrderFilter) (model.OrderListResult, error) {
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

func (f *fakeRepo) ListSellerOrders(ctx context.Context, sellerID int64, filter model.OrderFilter) (model.OrderListResult, error) {
	if f.listSellerOrdersFn == nil {
		return model.OrderListResult{}, nil
	}
	return f.listSellerOrdersFn(ctx, sellerID, filter)
}

func (f *fakeRepo) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	if f.getByIDFn == nil {
		return nil, model.ErrNotFound
	}
	return f.getByIDFn(ctx, id)
}

func (f *fakeRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	if f.updateStatusFn == nil {
		return nil
	}
	return f.updateStatusFn(ctx, id, status)
}

func (f *fakeRepo) GetSellerOrder(ctx context.Context, sellerID int64, orderID int64) (*model.OrderDetail, error) {
	if f.getSellerOrderFn == nil {
		return nil, model.ErrNotFound
	}
	return f.getSellerOrderFn(ctx, sellerID, orderID)
}

func setupRouter(repo *fakeRepo) *gin.Engine {
	handler := NewHandler(orderservice.NewService(repo))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(7))
		c.Next()
	})
	r.POST("/orders", handler.Place)
	r.GET("/orders", handler.List)
	r.GET("/orders/:id", handler.Get)
	return r
}

func TestPlace(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		repoErr    error
		wantStatus int
	}{
		{"успешный заказ", `{"items":[{"product_id":1,"quantity":2}]}`, nil, http.StatusCreated},
		{"пустые items", `{"items":[]}`, nil, http.StatusBadRequest},
		{"битый json", `{`, nil, http.StatusBadRequest},
		{"не хватает stock", `{"items":[{"product_id":1,"quantity":999}]}`, model.ErrNoStock, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{
				placeOrderFn: func(context.Context, int64, []model.OrderLineInput) (*model.Order, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return &model.Order{ID: 1, Status: model.OrderStatusNew, Total: 900000}, nil
				},
			}

			req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			setupRouter(repo).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestList(t *testing.T) {
	repo := &fakeRepo{
		listFn: func(context.Context, model.OrderFilter) (model.OrderListResult, error) {
			return model.OrderListResult{
				Items: []model.Order{{ID: 1, UserID: 7, Status: model.OrderStatusNew}},
				Total: 1, Page: 1, Limit: 20,
			}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/orders?status=new", nil)
	w := httptest.NewRecorder()
	setupRouter(repo).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestList_InvalidPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/orders?page=abc", nil)
	w := httptest.NewRecorder()
	setupRouter(&fakeRepo{}).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		order      *model.OrderDetail
		repoErr    error
		wantStatus int
	}{
		{
			name:       "свой заказ",
			url:        "/orders/1",
			order:      &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "чужой заказ — 403",
			url:        "/orders/1",
			order:      &model.OrderDetail{Order: model.Order{ID: 1, UserID: 999}},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "заказа нет — 404",
			url:        "/orders/999",
			repoErr:    model.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "id не число",
			url:        "/orders/abc",
			wantStatus: http.StatusBadRequest,
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

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			setupRouter(repo).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestList_NoUserID(t *testing.T) {
	handler := NewHandler(orderservice.NewService(&fakeRepo{}))

	r := gin.New()
	r.GET("/orders", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
