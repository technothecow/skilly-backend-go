package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"skilly/internal/domain/repository"
	"skilly/internal/infrastructure/gen"
)

func (s *Server) GetCheckUsername(c *gin.Context, params gen.GetCheckUsernameParams) {
	repo := repository.NewUserRepository(s.deps.Mongo, s.deps.Logger)
	_, err := repo.GetUserByUsername(c.Request.Context(), params.Username)

	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusOK, gen.CheckUsernameResponse{Available: true})
		} else {
			c.JSON(http.StatusInternalServerError, gen.CheckUsernameResponse{Available: false})
		}
		return
	}

	c.JSON(http.StatusOK, gen.CheckUsernameResponse{Available: false})
}
