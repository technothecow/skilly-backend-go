package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/oapi-codegen/gin-middleware"

	"skilly/internal/infrastructure/dependencies"
	"skilly/internal/infrastructure/gen"
	"skilly/internal/infrastructure/middleware"
	"skilly/internal/infrastructure/server"
	"skilly/internal/infrastructure/workers"
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

func startServer(deps *dependencies.Dependencies) func(ctx context.Context) {
	router := setupRouter()
	appServer := server.NewServer(deps)

	gen.RegisterHandlers(router, appServer)

	srv := &http.Server{
		Addr:    "0.0.0.0:8000",
		Handler: router,
	}

	// Start server in a goroutine so it doesn't block.
	go func() {
		deps.Logger.Info("Listening on port 8000...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			deps.Logger.Error("Failed to start server:", slog.Any("error", err))
		}
	}()

	return func (ctx context.Context) {
		if err := srv.Shutdown(ctx); err != nil {
			deps.Logger.Error("Server forced to shutdown:", slog.Any("error", err))
		}
	}
}

func main() {
	deps := dependencies.MustNewDependencies()

	workerManager := workers.NewWorkerManager(deps)
	workerManager.Start()

	stopServer := startServer(deps)

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until a signal is received
	deps.Logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	
	go stopServer(ctx)
	go workerManager.Stop()

	deps.Logger.Info("Server exiting")
}
