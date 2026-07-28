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
		min, max := int64(1000), int64(100)

		_, err := NewService(repo).List(context.Background(), model.ProductFilter{
			PriceMin: &min,
			PriceMax: &max,
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
