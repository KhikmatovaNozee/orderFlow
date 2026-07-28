package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	service := NewJWTService("test-secret")
	user := &model.User{ID: 42, Role: "seller"}

	token, err := service.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("сгенерировать токен: %v", err)
	}

	claims, err := service.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("разобрать токен: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("uid = %d, want %d", claims.UserID, user.ID)
	}
	if claims.Role != user.Role {
		t.Errorf("role = %q, want %q", claims.Role, user.Role)
	}
}

func TestParseAccessToken_Invalid(t *testing.T) {
	service := NewJWTService("test-secret")

	foreign, err := NewJWTService("another-secret").
		GenerateAccessToken(&model.User{ID: 1, Role: "user"})
	if err != nil {
		t.Fatalf("подготовить чужой токен: %v", err)
	}

	expired := signRaw(t, "test-secret", jwt.MapClaims{
		"uid":  float64(1),
		"role": "user",
		"exp":  time.Now().Add(-time.Hour).Unix(),
	})

	// Без роли — claims неполные.
	noRole := signRaw(t, "test-secret", jwt.MapClaims{
		"uid": float64(1),
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	tests := []struct {
		name  string
		token string
	}{
		{"пустая строка", ""},
		{"мусор вместо токена", "not-a-jwt"},
		{"чужая подпись", foreign},
		{"протухший", expired},
		{"нет роли в claims", noRole},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ParseAccessToken(tt.token)
			if !errors.Is(err, model.ErrInvalid) {
				t.Errorf("err = %v, want ErrInvalid", err)
			}
		})
	}
}

func signRaw(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("подписать токен: %v", err)
	}
	return signed
}
