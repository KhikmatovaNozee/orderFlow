package product

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	productservice "github.com/KhikmatovaNozee/orderFlow/internal/service/product"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeRepo struct {
	listFn    func(ctx context.Context, f model.ProductFilter) (model.ProductListResult, error)
	getByIDFn func(ctx context.Context, id int64) (*model.Product, error)
}

func (f *fakeRepo) List(ctx context.Context, filter model.ProductFilter) (model.ProductListResult, error) {
	if f.listFn == nil {
		return model.ProductListResult{}, nil
	}
	return f.listFn(ctx, filter)
}

func (f *fakeRepo) GetByID(ctx context.Context, id int64) (*model.Product, error) {
	if f.getByIDFn == nil {
		return nil, model.ErrNotFound
	}
	return f.getByIDFn(ctx, id)
}

func setupRouter(repo *fakeRepo) *gin.Engine {
	handler := NewHandler(productservice.NewService(repo))

	r := gin.New()
	r.GET("/products", handler.List)
	r.GET("/products/:id", handler.Get)
	return r
}

func TestList(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		listFn     func(context.Context, model.ProductFilter) (model.ProductListResult, error)
		wantStatus int
	}{
		{
			name: "успешный список",
			url:  "/products",
			listFn: func(context.Context, model.ProductFilter) (model.ProductListResult, error) {
				return model.ProductListResult{
					Items: []model.Product{{ID: 1, Name: "Наушники"}},
					Total: 1, Page: 1, Limit: 20,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "невалидный price_min",
			url:        "/products?price_min=abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "price_min больше price_max — ошибка от сервиса",
			url:        "/products?price_min=1000&price_max=100",
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

func TestGet(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		product    *model.Product
		repoErr    error
		wantStatus int
	}{
		{
			name:       "товар найден",
			url:        "/products/1",
			product:    &model.Product{ID: 1, Status: model.ProductStatusActive},
			wantStatus: http.StatusOK,
		},
		{
			name:       "скрытый товар — 404",
			url:        "/products/1",
			product:    &model.Product{ID: 1, Status: model.ProductStatusHidden},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "товара нет",
			url:        "/products/999",
			repoErr:    model.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "id не число",
			url:        "/products/abc",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{
				getByIDFn: func(context.Context, int64) (*model.Product, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return tt.product, nil
				},
			}
			r := setupRouter(repo)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
