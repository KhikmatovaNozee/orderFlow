package respond

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func TestError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{"not found", model.ErrNotFound, 404, `{"error":"entity not found"}`},
		{"invalid", model.ErrInvalid, 400, `{"error":"invalid input"}`},
		{"forbidden", model.ErrForbidden, 403, `{"error":"access forbidden"}`},
		{"no stock", model.ErrNoStock, 409, `{"error":"insufficient stock"}`},
		{"conflict", model.ErrConflict, 409, `{"error":"entity already exists"}`},

		{"wrapped", fmt.Errorf("create user: %w", model.ErrConflict), 409, `{"error":"entity already exists"}`},

		{"unknown", errors.New(`pq: relation "users" does not exist`), 500, `{"error":"internal server error"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newContext()

			Error(c, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if got := w.Body.String(); got != tt.wantBody {
				t.Errorf("body = %s, want %s", got, tt.wantBody)
			}
		})
	}
}

func TestErrorWithMessage(t *testing.T) {
	t.Run("своё сообщение вместо текста sentinel", func(t *testing.T) {
		c, w := newContext()

		ErrorWithMessage(c, model.ErrConflict, "login already exists")

		if w.Code != 409 {
			t.Errorf("status = %d, want 409", w.Code)
		}
		if got, want := w.Body.String(), `{"error":"login already exists"}`; got != want {
			t.Errorf("body = %s, want %s", got, want)
		}
	})

	t.Run("на 500 своё сообщение игнорируется", func(t *testing.T) {
		c, w := newContext()

		ErrorWithMessage(c, errors.New("pq: connection refused"), "не смогли достучаться до users")

		if w.Code != 500 {
			t.Errorf("status = %d, want 500", w.Code)
		}
		if got, want := w.Body.String(), `{"error":"internal server error"}`; got != want {
			t.Errorf("body = %s, want %s", got, want)
		}
	})
}

func TestJSON(t *testing.T) {
	type product struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	t.Run("отдаёт данные с нужным кодом", func(t *testing.T) {
		c, w := newContext()

		JSON(c, http.StatusCreated, product{ID: 7, Name: "bread"})

		if w.Code != 201 {
			t.Errorf("status = %d, want 201", w.Code)
		}
		if got, want := w.Body.String(), `{"id":7,"name":"bread"}`; got != want {
			t.Errorf("body = %s, want %s", got, want)
		}
	})

	t.Run("204 отдаётся без тела", func(t *testing.T) {
		c, w := newContext()

		JSON(c, http.StatusNoContent, nil)

		c.Writer.WriteHeaderNow()

		if w.Code != 204 {
			t.Errorf("status = %d, want 204", w.Code)
		}
		if w.Body.Len() != 0 {
			t.Errorf("body = %q, want empty", w.Body.String())
		}
	})
}
