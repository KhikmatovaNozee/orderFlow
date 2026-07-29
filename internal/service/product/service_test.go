package product

import (
	"context"
	"errors"
	"testing"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

type fakeRepo struct {
	listFn    func(ctx context.Context, f model.ProductFilter) (model.ProductListResult, error)
	getByIDFn func(ctx context.Context, id int64) (*model.Product, error)
	createFn  func(ctx context.Context, p model.Product) (model.Product, error)
	updateFn  func(ctx context.Context, p model.Product) (model.Product, error)
	gotFilter model.ProductFilter
	gotUpdate model.Product
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
	f.gotUpdate = p
	if f.updateFn == nil {
		return p, nil
	}
	return f.updateFn(ctx, p)
}

func TestService_List(t *testing.T) {
	t.Run("всегда форсирует status=active, даже если фильтр пришёл с другим", func(t *testing.T) {
		repo := &fakeRepo{}
		hidden := model.ProductStatusHidden

		_, err := NewService(repo).List(context.Background(), model.ProductFilter{Status: &hidden})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}

		if repo.gotFilter.Status == nil || *repo.gotFilter.Status != model.ProductStatusActive {
			t.Errorf("в репозиторий ушёл status=%v, want active", repo.gotFilter.Status)
		}
	})

	t.Run("price_min больше price_max — ошибка валидации", func(t *testing.T) {
		repo := &fakeRepo{}
		priceMin, priceMax := int64(1000), int64(100)

		_, err := NewService(repo).List(context.Background(), model.ProductFilter{
			PriceMin: &priceMin,
			PriceMax: &priceMax,
		})

		if !errors.Is(err, model.ErrInvalid) {
			t.Errorf("err = %v, want ErrInvalid", err)
		}
	})

	t.Run("пробрасывает ошибку репозитория как есть", func(t *testing.T) {
		repo := &fakeRepo{
			listFn: func(context.Context, model.ProductFilter) (model.ProductListResult, error) {
				return model.ProductListResult{}, errors.New("db is down")
			},
		}

		_, err := NewService(repo).List(context.Background(), model.ProductFilter{})
		if err == nil {
			t.Fatal("ожидали ошибку")
		}
	})
}

func TestService_GetByID(t *testing.T) {
	tests := []struct {
		name    string
		product *model.Product
		repoErr error
		wantErr error
	}{
		{
			name:    "активный товар возвращается",
			product: &model.Product{ID: 1, Status: model.ProductStatusActive},
		},
		{
			name:    "скрытый товар — как будто не найден",
			product: &model.Product{ID: 1, Status: model.ProductStatusHidden},
			wantErr: model.ErrNotFound,
		},
		{
			name:    "товара нет вообще",
			repoErr: model.ErrNotFound,
			wantErr: model.ErrNotFound,
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

			p, err := NewService(repo).GetByID(context.Background(), 1)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if p.Status != model.ProductStatusActive {
				t.Errorf("status = %s, want active", p.Status)
			}
		})
	}
}

func TestService_Create(t *testing.T) {
	tests := []struct {
		name    string
		in      model.CreateProductInput
		wantErr error
	}{
		{
			name: "успешное создание",
			in:   model.CreateProductInput{Name: "Наушники", Price: 1000, Stock: 5},
		},
		{
			name:    "пустое имя — 400",
			in:      model.CreateProductInput{Name: "  ", Price: 1000, Stock: 5},
			wantErr: model.ErrInvalid,
		},
		{
			name:    "price = 0 — 400",
			in:      model.CreateProductInput{Name: "Наушники", Price: 0, Stock: 5},
			wantErr: model.ErrInvalid,
		},
		{
			name:    "price < 0 — 400",
			in:      model.CreateProductInput{Name: "Наушники", Price: -10, Stock: 5},
			wantErr: model.ErrInvalid,
		},
		{
			name:    "stock < 0 — 400",
			in:      model.CreateProductInput{Name: "Наушники", Price: 1000, Stock: -1},
			wantErr: model.ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			p, err := NewService(repo).Create(context.Background(), 42, tt.in)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if p.SellerID != 42 {
				t.Errorf("seller_id = %d, want 42 (должен браться из аргумента, а не из тела)", p.SellerID)
			}
			if p.Status != model.ProductStatusActive {
				t.Errorf("status = %s, want active", p.Status)
			}
		})
	}
}

func TestService_Update(t *testing.T) {
	t.Run("чужой товар — 403", func(t *testing.T) {
		repo := &fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 999}, nil
			},
		}

		_, err := NewService(repo).Update(context.Background(), 42, 1, model.UpdateProductInput{})
		if !errors.Is(err, model.ErrForbidden) {
			t.Fatalf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("товара нет — 404", func(t *testing.T) {
		repo := &fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return nil, model.ErrNotFound
			},
		}

		_, err := NewService(repo).Update(context.Background(), 42, 1, model.UpdateProductInput{})
		if !errors.Is(err, model.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("невалидный price — 400", func(t *testing.T) {
		repo := &fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 42, Name: "Наушники", Price: 1000}, nil
			},
		}
		badPrice := int64(0)

		_, err := NewService(repo).Update(context.Background(), 42, 1, model.UpdateProductInput{Price: &badPrice})
		if !errors.Is(err, model.ErrInvalid) {
			t.Fatalf("err = %v, want ErrInvalid", err)
		}
	})

	t.Run("успешное обновление статуса active→hidden", func(t *testing.T) {
		repo := &fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 42, Name: "Наушники", Price: 1000, Status: model.ProductStatusActive}, nil
			},
		}
		hidden := model.ProductStatusHidden

		p, err := NewService(repo).Update(context.Background(), 42, 1, model.UpdateProductInput{Status: &hidden})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if p.Status != model.ProductStatusHidden {
			t.Errorf("status = %s, want hidden", p.Status)
		}
	})
}

func TestService_Delete(t *testing.T) {
	t.Run("чужой товар — 403", func(t *testing.T) {
		repo := &fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 999}, nil
			},
		}

		err := NewService(repo).Delete(context.Background(), 42, 1)
		if !errors.Is(err, model.ErrForbidden) {
			t.Fatalf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("успешное удаление переводит в hidden, не стирает строку", func(t *testing.T) {
		repo := &fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 42, Status: model.ProductStatusActive}, nil
			},
		}

		if err := NewService(repo).Delete(context.Background(), 42, 1); err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if repo.gotUpdate.Status != model.ProductStatusHidden {
			t.Errorf("в репозиторий ушёл status=%s, want hidden", repo.gotUpdate.Status)
		}
	})
}

func TestService_UpdatePhoto(t *testing.T) {
	t.Run("чужой товар — 403", func(t *testing.T) {
		repo := &fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 999}, nil
			},
		}

		_, err := NewService(repo).UpdatePhoto(context.Background(), 42, 1, "/uploads/products/1.jpg")
		if !errors.Is(err, model.ErrForbidden) {
			t.Fatalf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("успешно сохраняет путь к фото", func(t *testing.T) {
		repo := &fakeRepo{
			getByIDFn: func(context.Context, int64) (*model.Product, error) {
				return &model.Product{ID: 1, SellerID: 42}, nil
			},
		}

		p, err := NewService(repo).UpdatePhoto(context.Background(), 42, 1, "/uploads/products/1.jpg")
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if p.PhotoPath == nil || *p.PhotoPath != "/uploads/products/1.jpg" {
			t.Errorf("photo_path = %v, want /uploads/products/1.jpg", p.PhotoPath)
		}
	})
}
