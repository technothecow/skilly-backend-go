package server

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"skilly/internal/domain/models"
	"skilly/internal/domain/repository"
	"skilly/internal/domain/usecases"
	"skilly/internal/infrastructure/gen"
	"skilly/internal/infrastructure/security"
)

func (s *Server) PostRegister(c *gin.Context) {
	body, err := BindJSONAndHandleError[gen.RegisterRequest](c, &s.deps)
	if err != nil {
		return
	}

	repo := repository.NewUserRepository(s.deps.Mongo, s.deps.Logger)
	_, err = repo.GetUserByUsername(c.Request.Context(), body.Username)
	if err == nil {
		c.JSON(http.StatusConflict, gen.Error{
			Code: "username_already_exists",
		})
		return
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		s.deps.Logger.Error("failed to register user", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gen.Error{
			Code: "internal_server_error",
		})
		return
	}

	body.Password, err = security.HashPassword(body.Password)
	if err != nil {
		s.deps.Logger.Error("failed to hash password", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gen.Error{
			Code: "internal_server_error",
		})
		return
	}

	user := models.User{
		Username:  body.Username,
		Password:  body.Password,
		Bio:       body.Bio,
		Teaching:  body.Teaching,
		Learning:  body.Learning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = usecases.RegisterUser(c.Request.Context(), repo, user)
	if err != nil {
		s.deps.Logger.Error("failed to register user", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gen.Error{
			Code: "internal_server_error",
		})
		return
	}

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

	c.Status(http.StatusCreated)
}
