package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"skilly/internal/domain/repository"
	"skilly/internal/infrastructure/gen"
	"skilly/internal/infrastructure/security"
)

func (s *Server) PostProfileEdit(c *gin.Context) {
	username, err := security.AuthUser(c, s.deps)
	if err != nil {
		return
	}

	repo := repository.NewUserRepository(s.deps.Mongo, s.deps.Logger)
	user, err := repo.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		s.deps.Logger.Error("failed to get user", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gen.Error{
			Code: "internal_server_error",
		})
		return
	}

	body, err := BindJSONAndHandleError[gen.ProfileEditRequest](c, &s.deps)
	if err != nil {
		return
	}

	if body.Bio != nil {
		user.Bio = *body.Bio
	}
	if body.Learning != nil {
		user.Learning = *body.Learning
	}
	if body.Password != nil {
		user.Password, err = security.HashPassword(*body.Password)
		if err != nil {
			s.deps.Logger.Error("failed to hash password", slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, gen.Error{
				Code: "internal_server_error",
			})
			return
		}
	}
	if body.Teaching != nil {
		user.Teaching = *body.Teaching
	}

	err = repo.UpdateUser(c.Request.Context(), *user)
	if err != nil {
		s.deps.Logger.Error("failed to update user", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gen.Error{
			Code: "internal_server_error",
		})
		return
	}

	c.JSON(http.StatusOK, gen.UserProfile{
		Username: user.Username,
		PictureUrl: "",
		Bio: user.Bio,
		Teaching: user.Teaching,
		Learning: user.Learning,
	})
}
