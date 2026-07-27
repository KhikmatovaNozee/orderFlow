package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/logger"
	"github.com/KhikmatovaNozee/orderFlow/internal/respond"
	"github.com/gin-gonic/gin"
)

const pingTimeout = 2 * time.Second

type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db Pinger
}

func NewHandler(db Pinger) *Handler {
	return &Handler{db: db}
}

type healthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

func (h *Handler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), pingTimeout)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		logger.From(c.Request.Context()).Error("health check failed", slog.Any("error", err))
		respond.JSON(c, http.StatusServiceUnavailable, healthResponse{
			Status: "unavailable",
			DB:     "down",
		})
		return
	}

	respond.JSON(c, http.StatusOK, healthResponse{
		Status: "ok",
		DB:     "up",
	})
}
