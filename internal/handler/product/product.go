package product

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/KhikmatovaNozee/orderFlow/internal/respond"
	productservice "github.com/KhikmatovaNozee/orderFlow/internal/service/product"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *productservice.Service
}

func NewHandler(service *productservice.Service) *Handler {
	return &Handler{service: service}
}

type productsResponse struct {
	Items []model.Product `json:"items"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

func (h *Handler) List(c *gin.Context) {
	filter, err := parseFilter(c)
	if err != nil {
		respond.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		respond.Error(c, err)
		return
	}

	respond.JSON(c, http.StatusOK, productsResponse{
		Items: result.Items,
		Total: result.Total,
		Page:  result.Page,
		Limit: result.Limit,
	})
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respond.Fail(c, http.StatusBadRequest, "invalid product id")
		return
	}

	p, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respond.Error(c, err)
		return
	}

	respond.JSON(c, http.StatusOK, p)
}

func parseFilter(c *gin.Context) (model.ProductFilter, error) {
	var f model.ProductFilter

	if q := c.Query("q"); q != "" {
		f.Query = &q
	}
	if category := c.Query("category"); category != "" {
		f.Category = &category
	}

	priceMin, err := parseOptionalInt64(c, "price_min")
	if err != nil {
		return f, errors.New("invalid price_min")
	}
	f.PriceMin = priceMin

	priceMax, err := parseOptionalInt64(c, "price_max")
	if err != nil {
		return f, errors.New("invalid price_max")
	}
	f.PriceMax = priceMax

	page, err := parseOptionalInt64(c, "page")
	if err != nil {
		return f, errors.New("invalid page")
	}
	if page != nil {
		f.Page = int(*page)
	}

	limit, err := parseOptionalInt64(c, "limit")
	if err != nil {
		return f, errors.New("invalid limit")
	}
	if limit != nil {
		f.Limit = int(*limit)
	}

	return f, nil
}

func parseOptionalInt64(c *gin.Context, key string) (*int64, error) {
	raw := c.Query(key)
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
