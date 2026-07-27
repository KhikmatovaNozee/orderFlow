package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KhikmatovaNozee/orderFlow/internal/logger"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return slog.New(slog.NewJSONHandler(buf, nil)), buf
}

func TestLogging(t *testing.T) {
	t.Run("генерит request_id и отдаёт его в заголовке", func(t *testing.T) {
		log, buf := newTestLogger()

		r := gin.New()
		r.Use(Logging(log))
		r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))

		id := w.Header().Get(RequestIDHeader)
		if id == "" {
			t.Fatal("в ответе нет заголовка с request_id")
		}

		var entry map[string]any
		if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry); err != nil {
			t.Fatalf("лог не разобрался как JSON: %v", err)
		}
		if entry["request_id"] != id {
			t.Errorf("request_id в логе = %v, в заголовке = %s", entry["request_id"], id)
		}
		if entry["status"] != float64(200) {
			t.Errorf("status в логе = %v, want 200", entry["status"])
		}
	})

	t.Run("переиспользует request_id из входящего заголовка", func(t *testing.T) {
		log, _ := newTestLogger()

		r := gin.New()
		r.Use(Logging(log))
		r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set(RequestIDHeader, "from-the-outside")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if got := w.Header().Get(RequestIDHeader); got != "from-the-outside" {
			t.Errorf("request_id = %q, want from-the-outside", got)
		}
	})

	t.Run("кладёт логгер в контекст запроса", func(t *testing.T) {
		log, buf := newTestLogger()

		r := gin.New()
		r.Use(Logging(log))
		r.GET("/ping", func(c *gin.Context) {
			logger.From(c.Request.Context()).Info("что-то важное")
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))

		id := w.Header().Get(RequestIDHeader)
		if !bytes.Contains(buf.Bytes(), []byte("что-то важное")) {
			t.Fatal("сообщение из хендлера не попало в лог")
		}
		if bytes.Count(buf.Bytes(), []byte(id)) < 2 {
			t.Errorf("request_id %s не приклеился к логу хендлера", id)
		}
	})
}
