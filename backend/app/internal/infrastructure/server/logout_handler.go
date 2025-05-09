package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) PostLogout(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("authToken", "", -1, "/", "", false, true)
	c.Status(http.StatusNoContent)
}
