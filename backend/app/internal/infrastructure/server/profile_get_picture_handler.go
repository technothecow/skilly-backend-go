package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"skilly/internal/infrastructure/gen"
	"skilly/internal/infrastructure/security"
)

func (s *Server) GetProfileGetPicture(c *gin.Context, params gen.GetProfileGetPictureParams) {
	_, err := security.AuthUser(c, s.deps)
	if err != nil {
		return
	}

	path := fmt.Sprintf("pfp/%s", params.Username)

	url, err := s.deps.S3.GenerateDownloadUrl(c.Request.Context(), path, time.Minute*15)
	if err != nil {
		s.deps.Logger.Error("failed to generate download url", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gen.Error{
			Code: "internal_server_error",
		})
		return
	}

	c.JSON(http.StatusOK, gen.GetPictureResponse{
		Url: url.String(),
	})
}