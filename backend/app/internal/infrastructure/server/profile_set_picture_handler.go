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

func (s *Server) PostProfileSetPicture(c *gin.Context) {
	username, err := security.AuthUser(c, s.deps)
	if err != nil {
		return
	}

	url, err := s.deps.S3.GenerateUploadUrl(c.Request.Context(), fmt.Sprintf("pfp/%s", username), time.Minute*15)
	if err != nil {
		s.deps.Logger.Error("failed to generate upload url", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gen.Error{
			Code: "internal_server_error",
		})
		return
	}

	c.JSON(http.StatusOK, gen.SetPictureResponse{
		Url: url.String(),
	})
}