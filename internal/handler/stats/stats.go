package stats

import (
	"net/http"

	"github.com/KhikmatovaNozee/orderFlow/internal/respond"
	statservice "github.com/KhikmatovaNozee/orderFlow/internal/service/stats"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *statservice.Service
}

func NewHandler(service *statservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Get(c *gin.Context) {
	v, exists := c.Get("user_id")
	if !exists {
		respond.Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	sellerID := v.(int64)
	result, err := h.service.Get(c.Request.Context(), sellerID)

	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, result)
}
