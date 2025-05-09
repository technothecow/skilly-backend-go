package server

import (
	"context"
	"log/slog"
	"net/http"

	"skilly/internal/infrastructure/dependencies"
	"skilly/internal/infrastructure/gen"

	"github.com/gin-gonic/gin"
)

type Server struct {
	gen.ServerInterface
	deps dependencies.Dependencies
}

func NewServer() *Server {
	return &Server{deps: dependencies.SetupDependencies()}
}

func (s *Server) Shutdown(ctx context.Context) {
	s.deps.Logger.Info("Disconnecting MongoDB client...")
	s.deps.Mongo.Disconnect(ctx)
}

func BindJSONAndHandleError[T any](c *gin.Context, deps *dependencies.Dependencies) (T, error) {
	var body T
	if err := c.ShouldBindJSON(&body); err != nil {
		deps.Logger.Error("failed to bind JSON body", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gen.Error{
			Code:    "bad_request",
		})
		return body, err
	}
	return body, nil
}