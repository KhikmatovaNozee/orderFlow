package order

import (
	"net/http"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/KhikmatovaNozee/orderFlow/internal/respond"
	orderservice "github.com/KhikmatovaNozee/orderFlow/internal/service/order"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *orderservice.Service
}

func NewHandler(service *orderservice.Service) *Handler {
	return &Handler{service: service}
}

type placeOrderRequest struct {
	Items []struct {
		ProductID int64 `json:"product_id"`
		Quantity  int64 `json:"quantity"`
	} `json:"items"`
}

func (h *Handler) Place(c *gin.Context) {
	var req placeOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, ok := userIDFromContext(c)
	if !ok {
		respond.Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	items := make([]model.OrderLineInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, model.OrderLineInput{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
		})
	}

	order, err := h.service.Place(c.Request.Context(), userID, items)
	if err != nil {
		respond.Error(c, err)
		return
	}

	respond.JSON(c, http.StatusCreated, order)
}

func userIDFromContext(c *gin.Context) (int64, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}
