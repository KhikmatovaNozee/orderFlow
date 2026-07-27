package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/logger"
	"github.com/gin-gonic/gin"
)

const RequestIDHeader = "X-Request-Id"

const ContextKeyRequestID = "request_id"

func Logging(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = newRequestID()
		}

		c.Set(ContextKeyRequestID, requestID)
		c.Header(RequestIDHeader, requestID)

		reqLog := log.With(slog.String("request_id", requestID))
		c.Request = c.Request.WithContext(logger.Into(c.Request.Context(), reqLog))

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		status := c.Writer.Status()
		attrs := []any{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Duration("duration", duration),
		}

		switch {
		case status >= 500:
			reqLog.Error("request failed", attrs...)
		case status >= 400:
			reqLog.Warn("request rejected", attrs...)
		default:
			reqLog.Info("request handled", attrs...)
		}
	}
}

func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get(ContextKeyRequestID); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
