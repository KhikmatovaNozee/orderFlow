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
	getSellerOrderFn   func(ctx context.Context, sellerID int64, orderID int64) (*model.OrderDetail, error)
	payFn              func(ctx context.Context, id int64) (*model.Order, error)
	cancelFn           func(ctx context.Context, id int64) (*model.Order, error)
	shipFn             func(ctx context.Context, id int64, tracking string) (*model.Order, error)
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

func (f *fakeRepo) GetSellerOrder(ctx context.Context, sellerID int64, orderID int64) (*model.OrderDetail, error) {
	if f.getSellerOrderFn == nil {
		return nil, model.ErrNotFound
	}
	return f.getSellerOrderFn(ctx, sellerID, orderID)
}

func (f *fakeRepo) Ship(ctx context.Context, id int64, tracking string) (*model.Order, error) {
	if f.shipFn == nil {
		return nil, model.ErrInvalid
	}
	return f.shipFn(ctx, id, tracking)
}

func (f *fakeRepo) Pay(ctx context.Context, id int64) (*model.Order, error) {
	if f.payFn == nil {
		return nil, model.ErrInvalid
	}
	return f.payFn(ctx, id)
}

func (f *fakeRepo) Cancel(ctx context.Context, id int64) (*model.Order, error) {
	if f.cancelFn == nil {
		return nil, model.ErrInvalid
	}
	return f.cancelFn(ctx, id)
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
	r.PUT("/orders/:id/pay", handler.Pay)
	r.PUT("/orders/:id/cancel", handler.Cancel)
	r.GET("/manage/orders", handler.ListSellerOrders)
	r.GET("/manage/orders/:id", handler.GetSellerOrder)
	r.PUT("/manage/orders/:id/ship", handler.Ship)
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
		{"не хватает stock", `{"items":[{"product_id":1,"quantity":999}]}`, model.ErrNoStock, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{
				placeOrderFn: func(context.Context, int64, []model.OrderLineInput) (*model.Order, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return &model.Order{ID: 1, Status: model.OrderStatusNew}, nil
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

func TestGet(t *testing.T) {
	tests := []struct {
		name       string
		order      *model.OrderDetail
		repoErr    error
		wantStatus int
	}{
		{"свой заказ", &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7}}, nil, http.StatusOK},
		{"чужой заказ — 403", &model.OrderDetail{Order: model.Order{ID: 1, UserID: 999}}, nil, http.StatusForbidden},
		{"заказа нет — 404", nil, model.ErrNotFound, http.StatusNotFound},
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

			req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
			w := httptest.NewRecorder()
			setupRouter(repo).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestPay(t *testing.T) {
	tests := []struct {
		name       string
		order      *model.OrderDetail
		payErr     error
		wantStatus int
	}{
		{
			name:       "успешная оплата",
			order:      &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7, Status: model.OrderStatusNew}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "чужой заказ — 403",
			order:      &model.OrderDetail{Order: model.Order{ID: 1, UserID: 999, Status: model.OrderStatusNew}},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "уже оплачен — 400",
			order:      &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7, Status: model.OrderStatusPaid}},
			payErr:     model.ErrInvalid,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{
				getDetailFn: func(context.Context, int64) (*model.OrderDetail, error) {
					return tt.order, nil
				},
				payFn: func(context.Context, int64) (*model.Order, error) {
					if tt.payErr != nil {
						return nil, tt.payErr
					}
					return &model.Order{ID: 1, Status: model.OrderStatusPaid}, nil
				},
			}

			req := httptest.NewRequest(http.MethodPut, "/orders/1/pay", nil)
			w := httptest.NewRecorder()
			setupRouter(repo).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestCancel(t *testing.T) {
	tests := []struct {
		name       string
		order      *model.OrderDetail
		cancelErr  error
		wantStatus int
	}{
		{
			name:       "успешная отмена",
			order:      &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7, Status: model.OrderStatusNew}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "чужой заказ — 403",
			order:      &model.OrderDetail{Order: model.Order{ID: 1, UserID: 999, Status: model.OrderStatusNew}},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "уже отгружен — отменить нельзя, 400",
			order:      &model.OrderDetail{Order: model.Order{ID: 1, UserID: 7, Status: model.OrderStatusShipped}},
			cancelErr:  model.ErrInvalid,
			wantStatus: http.StatusBadRequest,
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

			req := httptest.NewRequest(http.MethodPut, "/orders/1/cancel", nil)
			w := httptest.NewRecorder()
			setupRouter(repo).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestShip(t *testing.T) {
	tests := []struct {
		name           string
		order          *model.OrderDetail
		sellerOrderErr error
		shipErr        error
		wantStatus     int
	}{
		{
			name:       "успешная отгрузка",
			order:      &model.OrderDetail{Order: model.Order{ID: 1, Status: model.OrderStatusPaid}},
			wantStatus: http.StatusOK,
		},
		{
			name:           "заказ не содержит товаров этого продавца — 404",
			sellerOrderErr: model.ErrNotFound,
			wantStatus:     http.StatusNotFound,
		},
		{
			name:       "заказ ещё не оплачен — 400",
			order:      &model.OrderDetail{Order: model.Order{ID: 1, Status: model.OrderStatusNew}},
			shipErr:    model.ErrInvalid,
			wantStatus: http.StatusBadRequest,
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
				shipFn: func(context.Context, int64, string) (*model.Order, error) {
					if tt.shipErr != nil {
						return nil, tt.shipErr
					}
					return &model.Order{
						ID:     1,
						Status: model.OrderStatusShipped,
					}, nil
				},
			}

			req := httptest.NewRequest(
				http.MethodPut, "/manage/orders/1/ship", strings.NewReader(`{"tracking":"TRACK123"}`))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			setupRouter(repo).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func setupRouterNoAuth(repo *fakeRepo) *gin.Engine {
	handler := NewHandler(orderservice.NewService(repo))

	r := gin.New()
	r.POST("/orders", handler.Place)
	r.GET("/orders", handler.List)
	r.GET("/orders/:id", handler.Get)
	r.PUT("/orders/:id/pay", handler.Pay)
	r.PUT("/orders/:id/cancel", handler.Cancel)
	r.GET("/manage/orders", handler.ListSellerOrders)
	r.GET("/manage/orders/:id", handler.GetSellerOrder)
	r.PUT("/manage/orders/:id/ship", handler.Ship)
	return r
}

func TestList(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		listFn     func(context.Context, model.OrderFilter) (model.OrderListResult, error)
		wantStatus int
	}{
		{
			name: "успешный список",
			url:  "/orders",
			listFn: func(context.Context, model.OrderFilter) (model.OrderListResult, error) {
				return model.OrderListResult{
					Items: []model.Order{{ID: 1, UserID: 7, Status: model.OrderStatusNew}},
					Total: 1, Page: 1, Limit: 20,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "фильтр по статусу",
			url:        "/orders?status=new&page=1&limit=10",
			wantStatus: http.StatusOK,
		},
		{
			name:       "невалидный page — 400",
			url:        "/orders?page=abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "невалидный limit — 400",
			url:        "/orders?limit=abc",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupRouter(&fakeRepo{listFn: tt.listFn})

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestListSellerOrders(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		listFn     func(context.Context, int64, model.OrderFilter) (model.OrderListResult, error)
		wantStatus int
	}{
		{
			name: "успешный список заказов продавца",
			url:  "/manage/orders",
			listFn: func(context.Context, int64, model.OrderFilter) (model.OrderListResult, error) {
				return model.OrderListResult{
					Items: []model.Order{{ID: 1, Status: model.OrderStatusPaid}},
					Total: 1, Page: 1, Limit: 20,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "фильтр по статусу",
			url:        "/manage/orders?status=paid",
			wantStatus: http.StatusOK,
		},
		{
			name:       "невалидный page — 400",
			url:        "/manage/orders?page=abc",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupRouter(&fakeRepo{listSellerOrdersFn: tt.listFn})

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestGetSellerOrder(t *testing.T) {
	tests := []struct {
		name       string
		order      *model.OrderDetail
		repoErr    error
		wantStatus int
	}{
		{
			name:       "заказ с товаром этого продавца",
			order:      &model.OrderDetail{Order: model.Order{ID: 1, Status: model.OrderStatusPaid}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "заказ без товаров этого продавца — 404",
			repoErr:    model.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{
				getSellerOrderFn: func(context.Context, int64, int64) (*model.OrderDetail, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return tt.order, nil
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/manage/orders/1", nil)
			w := httptest.NewRecorder()
			setupRouter(repo).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestUnauthorized(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		body   string
	}{
		{"Place", http.MethodPost, "/orders", `{"items":[{"product_id":1,"quantity":1}]}`},
		{"List", http.MethodGet, "/orders", ""},
		{"Get", http.MethodGet, "/orders/1", ""},
		{"Pay", http.MethodPut, "/orders/1/pay", ""},
		{"Cancel", http.MethodPut, "/orders/1/cancel", ""},
		{"ListSellerOrders", http.MethodGet, "/manage/orders", ""},
		{"GetSellerOrder", http.MethodGet, "/manage/orders/1", ""},
		{"Ship", http.MethodPut, "/manage/orders/1/ship", `{"tracking":"T1"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			setupRouterNoAuth(&fakeRepo{}).ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401, body=%s", w.Code, w.Body.String())
			}
		})
	}
}
