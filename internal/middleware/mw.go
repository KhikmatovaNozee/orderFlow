package middleware

import (
	"net/http"
	"strings"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/KhikmatovaNozee/orderFlow/internal/respond"
	"github.com/KhikmatovaNozee/orderFlow/internal/service/auth"
	"github.com/gin-gonic/gin"
)

func Auth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")

		if header == "" {
			respond.Fail(c, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(header, " ", 2)

		if len(parts) != 2 || parts[0] != "Bearer" {
			respond.Fail(c, http.StatusUnauthorized, "invalid authorization header")
			return
		}

		claims, err := jwtService.ParseAccessToken(parts[1])
		if err != nil {
			respond.Fail(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get("role")

		if !exists {
			respond.Error(c, model.ErrForbidden)
			return
		}

		userRole, ok := value.(string)
		if !ok || userRole != role {
			respond.Error(c, model.ErrForbidden)
			return
		}

		c.Next()
	}
}
