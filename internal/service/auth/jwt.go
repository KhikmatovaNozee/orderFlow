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

type AccessClaims struct {
	UserID int64
	Role   string
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

func (s *JWTService) ParseAccessToken(tokenString string) (*AccessClaims, error) {
	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}

			return []byte(s.secret), nil
		},
	)

	if err != nil || !token.Valid {
		return nil, model.ErrInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, model.ErrInvalid
	}

	userIDFloat, ok := claims["uid"].(float64)
	if !ok {
		return nil, model.ErrInvalid
	}

	role, ok := claims["role"].(string)
	if !ok {
		return nil, model.ErrInvalid
	}

	return &AccessClaims{
		UserID: int64(userIDFloat),
		Role:   role,
	}, nil
}
