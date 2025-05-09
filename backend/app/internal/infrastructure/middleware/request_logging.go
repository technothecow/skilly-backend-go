package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultMaxLogBodySize = 1024 // 1KB
)

// bodyLogWriter wraps gin.ResponseWriter to capture the response body.
type bodyLogWriter struct {
	gin.ResponseWriter
	bodyCopy *bytes.Buffer
}

// Write captures the data written to the response body.
func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.bodyCopy.Write(b) // Capture the body
	return w.ResponseWriter.Write(b)
}

// WriteString captures the string written to the response body.
func (w bodyLogWriter) WriteString(s string) (int, error) {
	w.bodyCopy.WriteString(s) // Capture the body
	return w.ResponseWriter.WriteString(s)
}

// LoggerConfig allows configuration of the RequestResponseLogger.
type LoggerConfig struct {
	MaxLogBodySize    int
	ShouldLogBodyFunc func(contentType string) bool // Custom function to decide if a body's content type is loggable
	Logger            *slog.Logger
}

// defaultShouldLogBody provides a default implementation for ShouldLogBodyFunc.
// It logs common text-based content types.
func defaultShouldLogBody(contentType string) bool {
	ctBase := strings.ToLower(strings.Split(contentType, ";")[0]) // Get base MIME type, e.g., "application/json" from "application/json; charset=utf-8"

	switch ctBase {
	case "application/json",
		"application/xml",
		"application/x-www-form-urlencoded":
		return true
	default:
		// Check for 'text/*' family (e.g., text/plain, text/html)
		return strings.HasPrefix(ctBase, "text/")
	}
}

// RequestResponseLogger logs incoming HTTP request and outgoing response details, including bodies.
func RequestResponseLogger(config ...LoggerConfig) gin.HandlerFunc {
	cfg := LoggerConfig{
		MaxLogBodySize:    defaultMaxLogBodySize,
		ShouldLogBodyFunc: defaultShouldLogBody,
		Logger:            slog.Default(),
	}
	if len(config) > 0 {
		if config[0].MaxLogBodySize > 0 {
			cfg.MaxLogBodySize = config[0].MaxLogBodySize
		}
		if config[0].ShouldLogBodyFunc != nil {
			cfg.ShouldLogBodyFunc = config[0].ShouldLogBodyFunc
		}
		if config[0].Logger != nil {
			cfg.Logger = config[0].Logger
		}
	}

	return func(c *gin.Context) {
		startTime := time.Now()
		var requestBodyString string

		// 1. Capture Request Body
		// Check if there's a body and it's not http.NoBody (which Gin sets for GET etc.)
		if c.Request.Body != nil && c.Request.Body != http.NoBody {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				cfg.Logger.Warn("Error reading request body", slog.Any("error", err))
				// Even if read fails, replace body with an empty reader so chain isn't broken
				c.Request.Body = io.NopCloser(bytes.NewBuffer(nil))
			} else {
				// IMPORTANT: Restore the request body so subsequent handlers can read it.
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				if len(bodyBytes) > 0 {
					contentType := c.GetHeader("Content-Type")
					if cfg.ShouldLogBodyFunc(contentType) {
						if len(bodyBytes) > cfg.MaxLogBodySize {
							requestBodyString = string(bodyBytes[:cfg.MaxLogBodySize]) + "... (truncated)"
						} else {
							requestBodyString = string(bodyBytes)
						}
					} else {
						requestBodyString = "[omitted: content type '" + contentType + "']"
					}
				}
			}
		}

		// 2. Prepare to capture response body
		// Wrap the original ResponseWriter
		blw := &bodyLogWriter{bodyCopy: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// 3. Process request (call subsequent handlers)
		c.Next()

		// 4. After request, capture response details and log
		latency := time.Since(startTime)
		statusCode := blw.Status() // Get status from our writer
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.RequestURI

		var responseBodyString string
		responseBodyBytes := blw.bodyCopy.Bytes()
		if len(responseBodyBytes) > 0 {
			contentType := blw.Header().Get("Content-Type") // Get content type from response header
			if cfg.ShouldLogBodyFunc(contentType) {
				if len(responseBodyBytes) > cfg.MaxLogBodySize {
					responseBodyString = string(responseBodyBytes[:cfg.MaxLogBodySize]) + "... (truncated)"
				} else {
					responseBodyString = string(responseBodyBytes)
				}
			} else {
				responseBodyString = "[omitted: content type '" + contentType + "']"
			}
		}

		var logBuilder strings.Builder
		logBuilder.WriteString(startTime.Format("2006/01/02 - 15:04:05"))
		logBuilder.WriteString(" | " + strconv.Itoa(statusCode))
		logBuilder.WriteString(" | " + latency.String())
		logBuilder.WriteString(" | " + clientIP)
		logBuilder.WriteString(" | " + method + " " + path)

		cfg.Logger.Info(
			logBuilder.String(), 
			slog.String("request_body", requestBodyString), 
			slog.String("response_body", responseBodyString), 
			slog.Any("errors", c.Errors.String()),
		)
	}
}
