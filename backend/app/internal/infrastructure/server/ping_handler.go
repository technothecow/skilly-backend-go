package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"skilly/internal/infrastructure/gen"
)

func (*Server) GetPing(ctx *gin.Context) {
	resp := gen.PongResponse{
		Message: "pong",
	}

	ctx.JSON(http.StatusOK, resp)
}
