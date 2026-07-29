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
	placeOrderFn func(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error)
}

func (f *fakeRepo) PlaceOrder(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error) {
	return f.placeOrderFn(ctx, userID, items)
}

func setupRouter(repo *fakeRepo) *gin.Engine {
	handler := NewHandler(orderservice.NewService(repo))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(7))
		c.Next()
	})
	r.POST("/orders", handler.Place)
	return r
}

func TestPlace(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		repoErr    error
		wantStatus int
	}{
		{
			name:       "успешный заказ",
			body:       `{"items":[{"product_id":1,"quantity":2}]}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "пустые items",
			body:       `{"items":[]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "битый json",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "не хватает stock",
			body:       `{"items":[{"product_id":1,"quantity":999}]}`,
			repoErr:    model.ErrNoStock,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "hidden товар",
			body:       `{"items":[{"product_id":1,"quantity":1}]}`,
			repoErr:    model.ErrInvalid,
			wantStatus: http.StatusBadRequest,
		},
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

func TestPlace_NoUserID(t *testing.T) {
	handler := NewHandler(orderservice.NewService(&fakeRepo{}))

	r := gin.New()
	r.POST("/orders", handler.Place)

	req := httptest.NewRequest(http.MethodPost, "/orders",
		strings.NewReader(`{"items":[{"product_id":1,"quantity":1}]}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
