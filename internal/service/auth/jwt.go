package auth

import (
	"fmt"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{
		secret: []byte(secret),
	}
}

func (s *JWTService) GenerateAccessToken(user *model.User) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"uid":  user.ID,
		"role": user.Role,
		"exp":  now.Add(15 * time.Minute).Unix(),
		"iat":  now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return signedToken, nil
}
