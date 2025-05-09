package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/oapi-codegen/gin-middleware"

	"skilly/internal/infrastructure/gen"
	"skilly/internal/infrastructure/middleware"
	"skilly/internal/infrastructure/server"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "Set-Cookie"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	validationMiddleware, err := ginmiddleware.OapiValidatorFromYamlFile("api/openapi.yaml")
	if err != nil {
		panic(err)
	}
	loggingMiddleware := middleware.RequestResponseLogger()

	r.Use(validationMiddleware)
	r.Use(loggingMiddleware)

	return r
}

func main() {
	router := setupRouter()
	appServer := server.NewServer()
	logger := slog.Default()

	gen.RegisterHandlers(router, appServer)

	srv := &http.Server{
		Addr:    "0.0.0.0:8000",
		Handler: router,
	}

	// Start server in a goroutine so it doesn't block.
	go func() {
		log.Println("Listening on port 8000...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until a signal is received
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown:", slog.Any("error", err))
	}

	appServer.Shutdown(ctx)

	logger.Info("Server exiting")
}
