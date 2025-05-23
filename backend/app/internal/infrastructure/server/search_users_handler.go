package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"skilly/internal/domain/repository"
	"skilly/internal/infrastructure/gen"
	"skilly/internal/infrastructure/security"
)

func (s *Server) PostSearch(c *gin.Context) {
	username, err := security.AuthUser(c, s.deps)
	if err != nil {
		return
	}

	repo := repository.NewUserRepository(s.deps.Mongo, s.deps.Logger)

	user, err := repo.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		return
	}

	body, err := BindJSONAndHandleError[gen.SearchRequest](c, s.deps)
	if err != nil {
		return
	}

	searchResultRaw, err := repo.SearchUsers(c.Request.Context(), user.Username, *body.Username, user.Teaching, *body.Skills, int64(*body.Page), int64(*body.Pagesize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gen.Error{
			Code: "internal_server_error",
		})
		return
	}

	searchResult := make([]gen.UserProfile, len(searchResultRaw))
	for i, user := range searchResultRaw {
		searchResult[i] = gen.UserProfile{
			Username: user.Username,
			Bio: user.Bio,
			Teaching: user.Teaching,
			Learning: user.Learning,
		}
	}

	c.JSON(http.StatusOK, gen.SearchResponse{Users: searchResult})
}