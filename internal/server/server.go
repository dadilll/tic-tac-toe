package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"tic_tac_toe/internal/handler"
	"tic_tac_toe/internal/router"
	"tic_tac_toe/pkg/logger"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
)

func New(rdb *redis.Client, Logger logger.Logger) *echo.Echo {
	e := echo.New()
	e.Validator = &handler.CustomValidator{Validator: validator.New()}
	router.SetupRoutes(e, rdb, Logger)

	return e
}

func Start(server *echo.Echo, Logger logger.Logger, port int) *http.Server {
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: server,
	}

	go func() {
		Logger.Info(context.Background(), fmt.Sprintf("Starting server on port :%d", port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			Logger.Error(context.Background(), "Failed to start server: "+err.Error())
		}
	}()

	return httpServer
}

func Stop(httpServer *http.Server, Logger logger.Logger) error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	Logger.Info(context.Background(), "Received shutdown signal, shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		Logger.Error(context.Background(), "Server Shutdown Failed: "+err.Error())
		return err
	}

	Logger.Info(context.Background(), "Server stopped gracefully")
	return nil
}
