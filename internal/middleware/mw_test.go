package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	authservice "github.com/KhikmatovaNozee/orderFlow/internal/service/auth"
	"github.com/gin-gonic/gin"
)

const testSecret = "test-secret"

func tokenFor(t *testing.T, jwtService *authservice.JWTService, role string) string {
	t.Helper()

	token, err := jwtService.GenerateAccessToken(&model.User{ID: 42, Role: role})
	if err != nil {
		t.Fatalf("сгенерировать токен: %v", err)
	}
	return token
}

func TestAuth(t *testing.T) {
	jwtService := authservice.NewJWTService(testSecret)
	validToken := tokenFor(t, jwtService, "user")

	foreign := tokenFor(t, authservice.NewJWTService("another-secret"), "user")

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"нет заголовка", "", http.StatusUnauthorized},
		{"без схемы Bearer", validToken, http.StatusUnauthorized},
		{"чужая схема", "Basic " + validToken, http.StatusUnauthorized},
		{"мусор вместо токена", "Bearer not-a-jwt", http.StatusUnauthorized},
		{"чужая подпись", "Bearer " + foreign, http.StatusUnauthorized},
		{"валидный токен", "Bearer " + validToken, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/protected", Auth(jwtService), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"user_id": c.MustGet("user_id"),
					"role":    c.MustGet("role"),
				})
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestAuth_SetsContextValues(t *testing.T) {
	jwtService := authservice.NewJWTService(testSecret)

	r := gin.New()
	r.GET("/protected", Auth(jwtService), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": c.MustGet("user_id"),
			"role":    c.MustGet("role"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, jwtService, "seller"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, `"user_id":42`) {
		t.Errorf("в контексте нет user_id: %s", body)
	}
	if !strings.Contains(body, `"role":"seller"`) {
		t.Errorf("в контексте нет роли: %s", body)
	}
}

func TestRequireRole(t *testing.T) {
	jwtService := authservice.NewJWTService(testSecret)

	tests := []struct {
		name       string
		role       string
		withAuth   bool
		wantStatus int
	}{
		{"роль совпала", "seller", true, http.StatusOK},
		{"роль не совпала", "user", true, http.StatusForbidden},
		{"без Auth впереди", "", false, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()

			handlers := []gin.HandlerFunc{}
			if tt.withAuth {
				handlers = append(handlers, Auth(jwtService))
			}
			handlers = append(handlers, RequireRole("seller"), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			r.GET("/manage", handlers...)

			req := httptest.NewRequest(http.MethodGet, "/manage", nil)
			if tt.withAuth {
				req.Header.Set("Authorization", "Bearer "+tokenFor(t, jwtService, tt.role))
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
