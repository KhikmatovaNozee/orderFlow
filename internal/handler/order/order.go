package order

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

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

type ordersResponse struct {
	Items []model.Order `json:"items"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		respond.Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	filter, err := parseOrderFilter(c)
	if err != nil {
		respond.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	role, _ := c.Get("role")
	var result model.OrderListResult

	if role == "seller" {
		result, err = h.service.ListSeller(c.Request.Context(), userID, filter)
	} else {
		result, err = h.service.List(c.Request.Context(), userID, filter)
	}
	if err != nil {
		respond.Error(c, err)
		return
	}

	respond.JSON(c, http.StatusOK, ordersResponse{
		Items: result.Items,
		Total: result.Total,
		Page:  result.Page,
		Limit: result.Limit,
	})
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		respond.Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	orderID, err := parseOrderID(c)
	if err != nil {
		respond.Fail(c, http.StatusBadRequest, "invalid order id")
		return
	}

	detail, err := h.service.GetDetail(c.Request.Context(), userID, orderID)
	if err != nil {
		respond.Error(c, err)
		return
	}

	respond.JSON(c, http.StatusOK, detail)
}

func (h *Handler) ListSellerOrders(c *gin.Context) {
	sellerID, ok := userIDFromContext(c)
	fmt.Println("SELLER ID FROM TOKEN:", sellerID)
func (h *Handler) Pay(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		respond.Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	filter, err := parseOrderFilter(c)
	if err != nil {
		respond.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.ListSellerOrders(c.Request.Context(), sellerID, filter)

	orderID, err := parseOrderID(c)
	if err != nil {
		respond.Fail(c, http.StatusBadRequest, "invalid order id")
		return
	}

	order, err := h.service.Pay(c.Request.Context(), userID, orderID)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.JSON(c, http.StatusOK, ordersResponse{
		Items: result.Items,
		Total: result.Total,
		Page:  result.Page,
		Limit: result.Limit,
	})
}

func (h *Handler) GetSellerOrder(c *gin.Context) {
	sellerID, ok := userIDFromContext(c)

	respond.JSON(c, http.StatusOK, order)
}

func (h *Handler) Cancel(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		respond.Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	orderID, err := parseOrderID(c)
	if err != nil {
		respond.Fail(c, http.StatusBadRequest, "invalid order id")
		return
	}

	detail, err := h.service.GetSellerOrder(c.Request.Context(), sellerID, orderID)
	order, err := h.service.Cancel(c.Request.Context(), userID, orderID)
	if err != nil {
		respond.Error(c, err)
		return
	}

	respond.JSON(c, http.StatusOK, detail)
	respond.JSON(c, http.StatusOK, order)
}

func parseOrderID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

func parseOrderFilter(c *gin.Context) (model.OrderFilter, error) {
	var f model.OrderFilter

	if status := c.Query("status"); status != "" {
		f.Status = &status
	}

	if raw := c.Query("page"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return f, errors.New("invalid page")
		}
		f.Page = v
	}
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return f, errors.New("invalid limit")
		}
		f.Limit = v
	}

	return f, nil
}

func userIDFromContext(c *gin.Context) (int64, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

func (h *Handler) Ship(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respond.Fail(c, 400, "invalid order id")
		return
	}

	err = h.service.Ship(c.Request.Context(), id)
	if err != nil {
		respond.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
