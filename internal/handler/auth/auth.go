package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/KhikmatovaNozee/orderFlow/internal/respond"
	"github.com/KhikmatovaNozee/orderFlow/internal/service/auth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *auth.Service
}

func NewHandler(service *auth.Service) *Handler {
	return &Handler{service: service}
}

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// userResponse — что отдаём наружу. Отдельная структура, а не model.User:
// у модели есть PasswordHash, и если однажды кто-то отдаст модель целиком,
// хеш утечёт. Явный тип это исключает на уровне компилятора.
type userResponse struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		// Доменной ошибки тут нет — тело запроса не распарсилось,
		// до сервиса дело не дошло. Поэтому явный код.
		respond.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Register(
		c.Request.Context(),
		req.Login,
		req.Password,
		req.Role)
	if err != nil {
		// Код подбирает общая таблица из respond (ErrInvalid→400,
		// ErrConflict→409, остальное→500), а текст — наш, понятный клиенту.
		switch {
		case errors.Is(err, model.ErrInvalid):
			respond.ErrorWithMessage(c, err, "invalid login, password or role")
		case errors.Is(err, model.ErrConflict):
			respond.ErrorWithMessage(c, err, "login already exists")
		default:
			respond.Error(c, err)
		}
		return
	}

	respond.JSON(c, http.StatusCreated, userResponse{
		ID:        user.ID,
		Login:     user.Login,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	})
}
