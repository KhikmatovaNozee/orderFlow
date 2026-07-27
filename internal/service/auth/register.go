package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 6

type UserRepository interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
	GetByLogin(ctx context.Context, login string) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	GetByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)

	Revoke(
		ctx context.Context,
		tokenHash string,
	) error
}

type Service struct {
	userRepo         UserRepository
	refreshTokenRepo RefreshTokenRepository
	jwtService       *JWTService
}

func NewService(userRepo UserRepository, refreshTokenRepo RefreshTokenRepository, jwtService *JWTService) *Service {
	return &Service{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtService:       jwtService,
	}
}

func (s *Service) Register(ctx context.Context, login string, password string, role string) (*model.User, error) {
	login = strings.TrimSpace(login)

	if login == "" || len(password) < minPasswordLength {
		return nil, model.ErrInvalid
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

func (s *Service) Login(ctx context.Context, login string, password string) (string, string, error) {
	user, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		return "", "", model.ErrInvalid
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", model.ErrInvalid
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, refreshTokenHash, err := generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	if err := s.refreshTokenRepo.Create(ctx, user.ID, refreshTokenHash, expiresAt); err != nil {
		return "", "", fmt.Errorf("save refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}
