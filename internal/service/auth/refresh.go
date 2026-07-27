package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

func (s *Service) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	oldHash := hashRefreshToken(refreshToken)

	oldToken, err := s.refreshTokenRepo.GetByHash(
		ctx,
		oldHash,
	)
	if err != nil {
		return "", "", model.ErrInvalid
	}

	if oldToken.RevokedAt != nil {
		return "", "", model.ErrInvalid
	}

	if time.Now().After(oldToken.ExpiresAt) {
		return "", "", model.ErrInvalid
	}

	if err := s.refreshTokenRepo.Revoke(ctx, oldHash); err != nil {
		return "", "", fmt.Errorf(
			"revoke old refresh token: %w",
			err,
		)
	}

	user, err := s.userRepo.GetByID(
		ctx,
		oldToken.UserID,
	)
	if err != nil {
		return "", "", model.ErrInvalid
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user)
	if err != nil {
		return "", "", fmt.Errorf(
			"generate access token: %w",
			err,
		)
	}

	newRefreshToken, newRefreshHash, err := generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	expiresAt := time.Now().Add(
		7 * 24 * time.Hour,
	)

	if err := s.refreshTokenRepo.Create(
		ctx,
		user.ID,
		newRefreshHash,
		expiresAt,
	); err != nil {
		return "", "", fmt.Errorf(
			"save new refresh token: %w",
			err,
		)
	}
	return accessToken, newRefreshToken, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashRefreshToken(refreshToken)

	return s.refreshTokenRepo.Revoke(ctx, tokenHash)
}
