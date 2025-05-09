package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"skilly/internal/domain/repository"
	"skilly/internal/infrastructure/gen"
)

func (s *Server) PostProfileView(c *gin.Context, params gen.PostProfileViewParams) {
	repo := repository.NewUserRepository(s.deps.Mongo, s.deps.Logger)

	user, err := repo.GetUserByUsername(c.Request.Context(), params.Username)
	if err != nil {
		if err == repository.ErrUserNotFound {
			c.JSON(http.StatusBadRequest, gen.Error{
				Code: "username_not_found",
			})
			return
		}
		s.deps.Logger.Error("failed to get user", slog.Any("error", err))
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