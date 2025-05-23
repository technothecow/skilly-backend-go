package server

import (
	"log/slog"
	"net/http"

	"skilly/internal/infrastructure/dependencies"
	"skilly/internal/infrastructure/gen"

	"github.com/gin-gonic/gin"
)

type Server struct {
	gen.ServerInterface
	deps *dependencies.Dependencies
}

func NewServer(deps *dependencies.Dependencies) *Server {
	return &Server{deps: deps}
}

func BindJSONAndHandleError[T any](c *gin.Context, deps *dependencies.Dependencies) (T, error) {
	var body T
	if err := c.ShouldBindJSON(&body); err != nil {
		deps.Logger.Error("failed to bind JSON body", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gen.Error{
			Code: "bad_request",
		})
		return body, err
	}
	return body, nil
}
