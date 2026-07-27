package respond

import (
	"errors"
	"net/http"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/gin-gonic/gin"
)

type errorBody struct {
	Error string `json:"error"`
}

const internalMessage = "internal server error"

var errorMapping = []struct {
	sentinel error
	status   int
}{
	{model.ErrNotFound, http.StatusNotFound},   // 404
	{model.ErrInvalid, http.StatusBadRequest},  // 400
	{model.ErrForbidden, http.StatusForbidden}, // 403
	{model.ErrNoStock, http.StatusConflict},    // 409
	{model.ErrConflict, http.StatusConflict},   // 409
}

func JSON(c *gin.Context, code int, data any) {
	if code == http.StatusNoContent || data == nil {
		c.Status(code)
		return
	}
	c.JSON(code, data)
}

func Error(c *gin.Context, err error) {
	status, message := resolve(err)
	Fail(c, status, message)
}

func ErrorWithMessage(c *gin.Context, err error, message string) {
	status, safeMessage := resolve(err)
	if status == http.StatusInternalServerError {
		message = safeMessage
	}
	Fail(c, status, message)
}

func Fail(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, errorBody{Error: message})
}

func Status(err error) int {
	status, _ := resolve(err)
	return status
}

func resolve(err error) (int, string) {
	if err == nil {
		return http.StatusInternalServerError, internalMessage
	}

	for _, m := range errorMapping {
		if errors.Is(err, m.sentinel) {
			return m.status, m.sentinel.Error()
		}
	}

	return http.StatusInternalServerError, internalMessage
}
