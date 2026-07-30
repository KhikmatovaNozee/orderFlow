package product

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
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
	createFn  func(ctx context.Context, p model.Product) (model.Product, error)
	updateFn  func(ctx context.Context, p model.Product) (model.Product, error)
	gotFilter model.ProductFilter
}

func (f *fakeRepo) List(ctx context.Context, filter model.ProductFilter) (model.ProductListResult, error) {
	f.gotFilter = filter
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

func (f *fakeRepo) Create(ctx context.Context, p model.Product) (model.Product, error) {
	if f.createFn == nil {
		return p, nil
	}
	return f.createFn(ctx, p)
}

func (f *fakeRepo) Update(ctx context.Context, p model.Product) (model.Product, error) {
	if f.updateFn == nil {
		return p, nil
	}
	return f.updateFn(ctx, p)
}

func setupRouter(repo *fakeRepo) *gin.Engine {
	handler := NewHandler(productservice.NewService(repo))

	r := gin.New()
	r.GET("/products", handler.List)
	r.GET("/products/:id", handler.Get)
	return r
}

func setupManageRouter(repo *fakeRepo, sellerID int64) *gin.Engine {
	handler := NewHandler(productservice.NewService(repo))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", sellerID)
		c.Next()
	})
	r.POST("/manage/products", handler.Create)
	r.GET("/manage/products", handler.ListMine)
	r.PUT("/manage/products/:id", handler.Update)
	r.DELETE("/manage/products/:id", handler.Delete)
	r.POST("/manage/products/:id/photo", handler.UploadPhoto)
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

func TestCreate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "успешное создание",
			body:       `{"name":"Наушники","price":1000,"stock":5}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "пустое имя — 400",
			body:       `{"name":"","price":1000,"stock":5}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "price <= 0 — 400",
			body:       `{"name":"Наушники","price":0,"stock":5}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "битый json — 400",
			body:       `{not json`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupManageRouter(&fakeRepo{}, 42)

			req := httptest.NewRequest(http.MethodPost, "/manage/products", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusCreated {
				var got model.Product
				if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
					t.Fatalf("не смогли распарсить ответ: %v", err)
				}
				if got.SellerID != 42 {
					t.Errorf("seller_id в ответе = %d, want 42 (взят из токена)", got.SellerID)
				}
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name       string
		sellerID   int64
		getByIDFn  func(context.Context, int64) (*model.Product, error)
		body       string
		wantStatus int
	}{
		{
			name:     "успешное обновление своего товара",
			sellerID: 42,
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 42, Name: "Наушники", Price: 1000}, nil
			},
			body:       `{"price":1500}`,
			wantStatus: http.StatusOK,
		},
		{
			name:     "чужой товар — 403",
			sellerID: 42,
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 999}, nil
			},
			body:       `{"price":1500}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:     "товара нет — 404",
			sellerID: 42,
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return nil, model.ErrNotFound
			},
			body:       `{"price":1500}`,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupManageRouter(&fakeRepo{getByIDFn: tt.getByIDFn}, tt.sellerID)

			req := httptest.NewRequest(http.MethodPut, "/manage/products/1", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name       string
		sellerID   int64
		getByIDFn  func(context.Context, int64) (*model.Product, error)
		wantStatus int
	}{
		{
			name:     "успешное удаление своего товара",
			sellerID: 42,
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 42, Status: model.ProductStatusActive}, nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:     "чужой товар — 403",
			sellerID: 42,
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 999}, nil
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupManageRouter(&fakeRepo{getByIDFn: tt.getByIDFn}, tt.sellerID)

			req := httptest.NewRequest(http.MethodDelete, "/manage/products/1", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestUploadPhoto(t *testing.T) {
	buildMultipart := func(filename string, content []byte) (*bytes.Buffer, string) {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		part, _ := w.CreateFormFile("photo", filename)
		_, _ = part.Write(content)
		_ = w.Close()
		return body, w.FormDataContentType()
	}

	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}

	t.Run("успешная загрузка jpg", func(t *testing.T) {
		body, contentType := buildMultipart("photo.jpg", jpegBytes)

		r := setupManageRouter(&fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 42}, nil
			},
		}, 42)

		req := httptest.NewRequest(http.MethodPost, "/manage/products/1/photo", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("не изображение (текстовый файл под видом .jpg) — 400", func(t *testing.T) {
		body, contentType := buildMultipart("photo.jpg", []byte("это не фото, а текст"))

		r := setupManageRouter(&fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 42}, nil
			},
		}, 42)

		req := httptest.NewRequest(http.MethodPost, "/manage/products/1/photo", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400, body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("недопустимое расширение (.gif) — 400", func(t *testing.T) {
		body, contentType := buildMultipart("photo.gif", jpegBytes)

		r := setupManageRouter(&fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 42}, nil
			},
		}, 42)

		req := httptest.NewRequest(http.MethodPost, "/manage/products/1/photo", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400, body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("чужой товар — 403", func(t *testing.T) {
		body, contentType := buildMultipart("photo.jpg", jpegBytes)

		r := setupManageRouter(&fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 999}, nil
			},
		}, 42)

		req := httptest.NewRequest(http.MethodPost, "/manage/products/1/photo", body)
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403, body=%s", w.Code, w.Body.String())
		}
	})
}

func TestListMine(t *testing.T) {
	repo := &fakeRepo{
		listFn: func(context.Context, model.ProductFilter) (model.ProductListResult, error) {
			return model.ProductListResult{
				Items: []model.Product{{ID: 1, SellerID: 42, Name: "Наушники"}},
				Total: 1, Page: 1, Limit: 20,
			}, nil
		},
	}
	r := setupManageRouter(repo, 42)

	req := httptest.NewRequest(http.MethodGet, "/manage/products", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	if repo.gotFilter.SellerID == nil || *repo.gotFilter.SellerID != 42 {
		t.Errorf("в репозиторий ушёл SellerID=%v, want 42", repo.gotFilter.SellerID)
	}
}

func TestListMine_InvalidPriceRange(t *testing.T) {
	r := setupManageRouter(&fakeRepo{}, 42)

	req := httptest.NewRequest(http.MethodGet, "/manage/products?price_min=1000&price_max=100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400, body=%s", w.Code, w.Body.String())
	}
}

func TestUpdate_InvalidStatus(t *testing.T) {
	r := setupManageRouter(&fakeRepo{
		getByIDFn: func(context.Context, int64) (*model.Product, error) {
			return &model.Product{ID: 1, SellerID: 42}, nil
		},
	}, 42)

	req := httptest.NewRequest(http.MethodPut, "/manage/products/1", bytes.NewBufferString(`{"status":"deleted"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400, body=%s", w.Code, w.Body.String())
	}
}

func TestDelete_NotFound(t *testing.T) {
	r := setupManageRouter(&fakeRepo{
		getByIDFn: func(context.Context, int64) (*model.Product, error) {
			return nil, model.ErrNotFound
		},
	}, 42)

	req := httptest.NewRequest(http.MethodDelete, "/manage/products/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404, body=%s", w.Code, w.Body.String())
	}
}

func TestUploadPhoto_TooLarge(t *testing.T) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	part, _ := w.CreateFormFile("photo", "big.jpg")
	// Заголовок jpeg + добивка нулями сверх лимита в 5MB.
	big := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 5<<20+1)...)
	_, _ = part.Write(big)
	_ = w.Close()

	r := setupManageRouter(&fakeRepo{
		getByIDFn: func(context.Context, int64) (*model.Product, error) {
			return &model.Product{ID: 1, SellerID: 42}, nil
		},
	}, 42)

	req := httptest.NewRequest(http.MethodPost, "/manage/products/1/photo", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}
