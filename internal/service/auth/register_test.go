package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func newService(userRepo UserRepository, tokenRepo RefreshTokenRepository) *Service {
	return NewService(userRepo, tokenRepo, NewJWTService("test-secret"))
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		password string
		role     string
		repoErr  error
		wantErr  error
	}{
		{
			name:     "покупатель регистрируется",
			login:    "sevara",
			password: "password123",
			role:     "user",
		},
		{
			name:     "продавец регистрируется",
			login:    "seller1",
			password: "password123",
			role:     "seller",
		},
		{
			name:     "пустой логин",
			login:    "   ",
			password: "password123",
			role:     "user",
			wantErr:  model.ErrInvalid,
		},
		{
			name:     "слишком короткий пароль",
			login:    "sevara",
			password: "12345",
			role:     "user",
			wantErr:  model.ErrInvalid,
		},
		{
			name:     "неизвестная роль",
			login:    "sevara",
			password: "password123",
			role:     "admin",
			wantErr:  model.ErrInvalid,
		},
		{
			name:     "логин уже занят",
			login:    "sevara",
			password: "password123",
			role:     "user",
			repoErr:  model.ErrConflict,
			wantErr:  model.ErrConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserRepo{
				createFn: func(_ context.Context, user *model.User) (*model.User, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					user.ID = 1
					return user, nil
				},
			}

			user, err := newService(repo, &fakeTokenRepo{}).
				Register(context.Background(), tt.login, tt.password, tt.role)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if user.Role != tt.role {
				t.Errorf("role = %q, want %q", user.Role, tt.role)
			}
		})
	}
}

func TestRegister_HashesPasswordAndTrimsLogin(t *testing.T) {
	repo := &fakeUserRepo{
		createFn: func(_ context.Context, user *model.User) (*model.User, error) {
			return user, nil
		},
	}

	_, err := newService(repo, &fakeTokenRepo{}).
		Register(context.Background(), "  sevara  ", "password123", "user")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if repo.created.Login != "sevara" {
		t.Errorf("login = %q, want sevara", repo.created.Login)
	}

	if repo.created.PasswordHash == "password123" {
		t.Fatal("пароль сохранён в открытом виде")
	}
	if err := bcrypt.CompareHashAndPassword(
		[]byte(repo.created.PasswordHash),
		[]byte("password123"),
	); err != nil {
		t.Errorf("хэш не совпал с исходным паролем: %v", err)
	}
}
