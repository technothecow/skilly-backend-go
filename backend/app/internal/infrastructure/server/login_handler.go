package server

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"skilly/internal/domain/repository"
	"skilly/internal/infrastructure/gen"
	"skilly/internal/infrastructure/security"
)

func (s *Server) PostLogin(c *gin.Context) {
	body, err := BindJSONAndHandleError[gen.LoginRequest](c, &s.deps)
	if err != nil {
		return
	}

	repo := repository.NewUserRepository(s.deps.Mongo, s.deps.Logger)

	user, err := repo.GetUserByUsername(c.Request.Context(), body.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusBadRequest, gen.Error{
				Code: "invalid_credentials",
			})
			return
		}
		s.deps.Logger.Error("failed to get user", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gen.Error{
			Code: "internal_server_error",
		})
		return
	}

	if security.VerifyPassword(body.Password, user.Password) {
		token, err := security.CreateToken(user.Username)
		if err != nil {
			s.deps.Logger.Error("failed to create auth token", slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, gen.Error{
				Code: "failed_to_issue_auth_token",
			})
			return
		}
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("authToken", token, int(security.TokenTTL.Seconds()), "/", "", false, true)
		c.Status(http.StatusOK)
	} else {
		c.JSON(http.StatusBadRequest, gen.Error{
			Code: "invalid_credentials",
		})
	}
}
