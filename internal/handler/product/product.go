package product

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

const (
	maxPhotoSize = 5 << 20 // 5 MB
	uploadDir    = "uploads/products"
)

type createProductRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Price    int64  `json:"price"`
	Stock    int64  `json:"stock"`
}

type updateProductRequest struct {
	Name     *string `json:"name"`
	Category *string `json:"category"`
	Price    *int64  `json:"price"`
	Stock    *int64  `json:"stock"`
	Status   *string `json:"status"`
}

func sellerIDFromContext(c *gin.Context) (int64, bool) {
	v, ok := c.MustGet("user_id").(int64)
	return v, ok
}

func (h *Handler) Create(c *gin.Context) {
	sellerID, ok := sellerIDFromContext(c)
	if !ok {
		respond.Error(c, model.ErrForbidden)
		return
	}

	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}

	p, err := h.service.Create(c.Request.Context(), sellerID, model.CreateProductInput{
		Name:     req.Name,
		Category: req.Category,
		Price:    req.Price,
		Stock:    req.Stock,
	})
	if err != nil {
		respond.Error(c, err)
		return
	}

	respond.JSON(c, http.StatusCreated, p)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respond.Fail(c, http.StatusBadRequest, "invalid product id")
		return
	}

	sellerID, ok := sellerIDFromContext(c)
	if !ok {
		respond.Error(c, model.ErrForbidden)
		return
	}

	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}

	p, err := h.service.Update(c.Request.Context(), sellerID, id, model.UpdateProductInput{
		Name:     req.Name,
		Category: req.Category,
		Price:    req.Price,
		Stock:    req.Stock,
		Status:   req.Status,
	})
	if err != nil {
		respond.Error(c, err)
		return
	}

	respond.JSON(c, http.StatusOK, p)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respond.Fail(c, http.StatusBadRequest, "invalid product id")
		return
	}

	sellerID, ok := sellerIDFromContext(c)
	if !ok {
		respond.Error(c, model.ErrForbidden)
		return
	}

	if err := h.service.Delete(c.Request.Context(), sellerID, id); err != nil {
		respond.Error(c, err)
		return
	}

	respond.JSON(c, http.StatusNoContent, nil)
}

func (h *Handler) UploadPhoto(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respond.Fail(c, http.StatusBadRequest, "invalid product id")
		return
	}

	sellerID, ok := sellerIDFromContext(c)
	if !ok {
		respond.Error(c, model.ErrForbidden)
		return
	}

	fileHeader, err := c.FormFile("photo")
	if err != nil {
		respond.Fail(c, http.StatusBadRequest, "photo file is required")
		return
	}

	if fileHeader.Size > maxPhotoSize {
		respond.Fail(c, http.StatusBadRequest, "photo is too large (max 5MB)")
		return
	}

	ext, ok := allowedPhotoExt(fileHeader.Filename)
	if !ok {
		respond.Fail(c, http.StatusBadRequest, "only jpg/png photos are allowed")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		respond.Fail(c, http.StatusInternalServerError, "internal server error")
		return
	}
	defer func() {
		_ = file.Close()
	}()

	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	if contentType != "image/jpeg" && contentType != "image/png" {
		respond.Fail(c, http.StatusBadRequest, "only jpg/png photos are allowed")
		return
	}

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		respond.Fail(c, http.StatusInternalServerError, "internal server error")
		return
	}

	filename := fmt.Sprintf("%d_%d%s", id, time.Now().UnixNano(), ext)
	dstPath := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(fileHeader, dstPath); err != nil {
		respond.Fail(c, http.StatusInternalServerError, "internal server error")
		return
	}

	publicPath := "/" + dstPath

	p, err := h.service.UpdatePhoto(c.Request.Context(), sellerID, id, publicPath)
	if err != nil {
		_ = os.Remove(dstPath)
		respond.Error(c, err)
		return
	}

	respond.JSON(c, http.StatusOK, p)
}

func allowedPhotoExt(filename string) (string, bool) {
	switch ext := strings.ToLower(filepath.Ext(filename)); ext {
	case ".jpg", ".jpeg", ".png":
		return ext, true
	default:
		return "", false
	}
}
