package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	authservice "github.com/KhikmatovaNozee/orderFlow/internal/service/auth"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeUserRepo struct {
	user    *model.User
	findErr error
	saveErr error
}

func (f *fakeUserRepo) Create(_ context.Context, u *model.User) (*model.User, error) {
	if f.saveErr != nil {
		return nil, f.saveErr
	}
	u.ID = 1
	return u, nil
}

func (f *fakeUserRepo) GetByLogin(context.Context, string) (*model.User, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.user, nil
}

func (f *fakeUserRepo) GetByID(context.Context, int64) (*model.User, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.user, nil
}

type fakeTokenRepo struct {
	token   *model.RefreshToken
	findErr error
}

func (f *fakeTokenRepo) Create(context.Context, int64, string, time.Time) error {
	return nil
}

func (f *fakeTokenRepo) GetByHash(context.Context, string) (*model.RefreshToken, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.token, nil
}

func (f *fakeTokenRepo) Revoke(context.Context, string) error {
	return nil
}

func setupRouter(userRepo *fakeUserRepo, tokenRepo *fakeTokenRepo) *gin.Engine {
	jwtService := authservice.NewJWTService("test-secret")
	service := authservice.NewService(userRepo, tokenRepo, jwtService)
	handler := NewHandler(service, jwtService)

	r := gin.New()
	r.POST("/register", handler.Register)
	r.POST("/login", handler.Login)
	r.POST("/refresh", handler.Refresh)
	r.POST("/logout", handler.Logout)
	return r
}

func doRequest(r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		saveErr    error
		wantStatus int
	}{
		{"успех", `{"login":"sevara","password":"password123","role":"user"}`, nil, http.StatusCreated},
		{"битый json", `{"login":`, nil, http.StatusBadRequest},
		{"короткий пароль", `{"login":"sevara","password":"123","role":"user"}`, nil, http.StatusBadRequest},
		{"логин занят", `{"login":"sevara","password":"password123","role":"user"}`, model.ErrConflict, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserRepo{saveErr: tt.saveErr}
			w := doRequest(setupRouter(repo, &fakeTokenRepo{}), "/register", tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestLogin(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	user := &model.User{ID: 7, Login: "sevara", PasswordHash: string(hash), Role: "user"}

	tests := []struct {
		name       string
		body       string
		findErr    error
		wantStatus int
	}{
		{"верный пароль", `{"login":"sevara","password":"password123"}`, nil, http.StatusOK},
		{"неверный пароль", `{"login":"sevara","password":"wrong"}`, nil, http.StatusUnauthorized},
		{"юзера нет", `{"login":"nobody","password":"password123"}`, model.ErrNotFound, http.StatusUnauthorized},
		{"битый json", `{`, nil, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserRepo{user: user, findErr: tt.findErr}
			w := doRequest(setupRouter(repo, &fakeTokenRepo{}), "/login", tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	validToken := &model.RefreshToken{UserID: 7, ExpiresAt: time.Now().Add(24 * time.Hour)}
	user := &model.User{ID: 7, Login: "sevara", Role: "user"}

	tests := []struct {
		name       string
		body       string
		token      *model.RefreshToken
		findErr    error
		wantStatus int
	}{
		{"валидный токен", `{"refresh":"token"}`, validToken, nil, http.StatusOK},
		{"токена нет", `{"refresh":"token"}`, nil, model.ErrNotFound, http.StatusUnauthorized},
		{"битый json", `{`, nil, nil, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &fakeUserRepo{user: user}
			tokenRepo := &fakeTokenRepo{token: tt.token, findErr: tt.findErr}

			w := doRequest(setupRouter(userRepo, tokenRepo), "/refresh", tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestLogout(t *testing.T) {
	w := doRequest(setupRouter(&fakeUserRepo{}, &fakeTokenRepo{}), "/logout", `{"refresh":"token"}`)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestLogout_BadJSON(t *testing.T) {
	w := doRequest(setupRouter(&fakeUserRepo{}, &fakeTokenRepo{}), "/logout", `{`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
