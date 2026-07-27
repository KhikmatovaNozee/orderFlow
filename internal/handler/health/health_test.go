package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestCheck(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "база отвечает",
			pingErr:    nil,
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok","db":"up"}`,
		},
		{
			name:       "база недоступна",
			pingErr:    errors.New("connection refused"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `{"status":"unavailable","db":"down"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/health", NewHandler(fakePinger{err: tt.pingErr}).Check)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if got := w.Body.String(); got != tt.wantBody {
				t.Errorf("body = %s, want %s", got, tt.wantBody)
			}
		})
	}
}
