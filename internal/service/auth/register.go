package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 6

type UserRepository interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
}

type Service struct {
	userRepo UserRepository
}

func NewService(userRepo UserRepository) *Service {
	return &Service{userRepo: userRepo}
}

func (s *Service) Register(ctx context.Context, login string, password string, role string) (*model.User, error) {
	login = strings.TrimSpace(login)

	if login == "" || len(login) < minPasswordLength {
		return nil, model.ErrNotFound
	}
	if role != "user" && role != "seller" {
		return nil, model.ErrInvalid
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Login:        login,
		PasswordHash: string(passwordHash),
		Role:         role,
	}
	return s.userRepo.Create(ctx, user)
}
