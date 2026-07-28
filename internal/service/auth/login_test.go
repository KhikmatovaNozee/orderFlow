package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func userWithPassword(t *testing.T, password string) *model.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("сгенерировать хэш: %v", err)
	}

	return &model.User{
		ID:           7,
		Login:        "sevara",
		PasswordHash: string(hash),
		Role:         "user",
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name     string
		password string
		userErr  error
		saveErr  error
		wantErr  error
	}{
		{
			name:     "верный пароль",
			password: "password123",
		},
		{
			name:     "неверный пароль",
			password: "wrong-password",
			wantErr:  model.ErrInvalid,
		},
		{
			name:     "пользователя нет",
			password: "password123",
			userErr:  model.ErrNotFound,
			wantErr:  model.ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := userWithPassword(t, "password123")

			userRepo := &fakeUserRepo{
				getByLoginFn: func(context.Context, string) (*model.User, error) {
					if tt.userErr != nil {
						return nil, tt.userErr
					}
					return user, nil
				},
			}
			tokenRepo := &fakeTokenRepo{}

			access, refresh, err := newService(userRepo, tokenRepo).
				Login(context.Background(), "sevara", tt.password)

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
				t.Fatal("оба токена должны быть непустыми")
			}
			if len(tokenRepo.createdHashes) != 1 {
				t.Fatalf("refresh сохранён %d раз, want 1", len(tokenRepo.createdHashes))
			}
			if tokenRepo.createdHashes[0] == refresh {
				t.Error("в базу записан сам refresh-токен вместо хэша")
			}
		})
	}
}

func TestLogin_DatabaseFailureIsNotErrInvalid(t *testing.T) {
	userRepo := &fakeUserRepo{
		getByLoginFn: func(context.Context, string) (*model.User, error) {
			return userWithPassword(t, "password123"), nil
		},
	}
	tokenRepo := &fakeTokenRepo{
		createFn: func(context.Context, int64, string, time.Time) error {
			return errors.New("db is down")
		},
	}

	_, _, err := newService(userRepo, tokenRepo).
		Login(context.Background(), "sevara", "password123")

	if err == nil {
		t.Fatal("ожидали ошибку сохранения")
	}
	if errors.Is(err, model.ErrInvalid) {
		t.Error("сбой базы не должен превращаться в ErrInvalid")
	}
}
