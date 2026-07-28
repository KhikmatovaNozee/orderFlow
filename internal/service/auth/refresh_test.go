package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

func TestRefresh(t *testing.T) {
	revokedAt := time.Now().Add(-time.Hour)

	tests := []struct {
		name    string
		stored  *model.RefreshToken
		findErr error
		userErr error
		wantErr error
	}{
		{
			name: "валидный токен обменивается",
			stored: &model.RefreshToken{
				UserID:    7,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
		},
		{
			name:    "токена нет в базе",
			findErr: model.ErrNotFound,
			wantErr: model.ErrInvalid,
		},
		{
			name: "токен уже отозван",
			stored: &model.RefreshToken{
				UserID:    7,
				ExpiresAt: time.Now().Add(24 * time.Hour),
				RevokedAt: &revokedAt,
			},
			wantErr: model.ErrInvalid,
		},
		{
			name: "токен протух",
			stored: &model.RefreshToken{
				UserID:    7,
				ExpiresAt: time.Now().Add(-time.Minute),
			},
			wantErr: model.ErrInvalid,
		},
		{
			name: "пользователь исчез",
			stored: &model.RefreshToken{
				UserID:    7,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			userErr: model.ErrNotFound,
			wantErr: model.ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &fakeUserRepo{
				getByIDFn: func(context.Context, int64) (*model.User, error) {
					if tt.userErr != nil {
						return nil, tt.userErr
					}
					return &model.User{ID: 7, Login: "sevara", Role: "user"}, nil
				},
			}
			tokenRepo := &fakeTokenRepo{
				getByHashFn: func(context.Context, string) (*model.RefreshToken, error) {
					if tt.findErr != nil {
						return nil, tt.findErr
					}
					return tt.stored, nil
				},
			}

			access, refresh, err := newService(userRepo, tokenRepo).
				Refresh(context.Background(), "some-refresh-token")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if access == "" || refresh == "" {
				t.Error("оба токена должны быть непустыми")
			}
		})
	}
}

func TestRefresh_RotatesToken(t *testing.T) {
	const oldToken = "old-refresh-token"

	userRepo := &fakeUserRepo{
		getByIDFn: func(context.Context, int64) (*model.User, error) {
			return &model.User{ID: 7, Login: "sevara", Role: "user"}, nil
		},
	}
	tokenRepo := &fakeTokenRepo{
		getByHashFn: func(context.Context, string) (*model.RefreshToken, error) {
			return &model.RefreshToken{
				UserID:    7,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}, nil
		},
	}

	_, newRefresh, err := newService(userRepo, tokenRepo).
		Refresh(context.Background(), oldToken)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if len(tokenRepo.revokedHashes) != 1 {
		t.Fatalf("старый токен отозван %d раз, want 1", len(tokenRepo.revokedHashes))
	}
	if tokenRepo.revokedHashes[0] != hashRefreshToken(oldToken) {
		t.Error("отозвали не тот токен")
	}
	if len(tokenRepo.createdHashes) != 1 {
		t.Fatalf("новый токен сохранён %d раз, want 1", len(tokenRepo.createdHashes))
	}
	if newRefresh == oldToken {
		t.Error("новый refresh совпал со старым — ротации не произошло")
	}
}

func TestLogout(t *testing.T) {
	const token = "some-refresh-token"

	tokenRepo := &fakeTokenRepo{}

	if err := newService(&fakeUserRepo{}, tokenRepo).
		Logout(context.Background(), token); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if len(tokenRepo.revokedHashes) != 1 {
		t.Fatalf("Revoke вызван %d раз, want 1", len(tokenRepo.revokedHashes))
	}
	if tokenRepo.revokedHashes[0] != hashRefreshToken(token) {
		t.Error("Revoke вызван не с хэшем токена")
	}
}
