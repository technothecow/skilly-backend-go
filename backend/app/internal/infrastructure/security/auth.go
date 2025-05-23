package security

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"skilly/internal/infrastructure/dependencies"
	"skilly/internal/infrastructure/gen"
)

func AuthUser(c *gin.Context, deps *dependencies.Dependencies) (string, error) {
	authCookie, err := c.Cookie(TokenCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			c.JSON(http.StatusUnauthorized, gen.Error{
				Code: "unauthorized",
			})
			return "", err
		}
		deps.Logger.Error("failed to get auth cookie", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gen.Error{
			Code: "internal_server_error",
		})
		return "", err
	}
	
	username, err := GetUsernameFromToken(authCookie)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			c.JSON(http.StatusUnauthorized, gen.Error{
				Code: "invalid_credentials",
			})
			return "", err
		}
		if errors.Is(err, ErrExpiredToken) {
			c.JSON(http.StatusUnauthorized, gen.Error{
				Code: "expired_token",
			})
			return "", err
		}
		deps.Logger.Error("failed to get auth cookie", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gen.Error{
			Code: "internal_server_error",
		})
		return "", err
	}

	return username, nil
}