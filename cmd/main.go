package main

import (
	"context"
	"os"
	"tic_tac_toe/internal/config"
	"tic_tac_toe/internal/server"
	redis "tic_tac_toe/pkg/db/redis"
	"tic_tac_toe/pkg/logger"
)

const serviceName = "boatservice"

func main() {
	ctx := context.Background()
	Logger := logger.New(serviceName)
	ctx = context.WithValue(ctx, logger.LoggerKey, Logger)

	cfg := config.New()
	if cfg == nil {
		Logger.Error(ctx, "ERROR: config is nil")
		os.Exit(1)
	}

	rdb, err := redis.New(cfg.ConfigRedis)
	if err != nil {
		Logger.Error(ctx, "redis connection error: "+err.Error())
		return
	}

	e := server.New(rdb, Logger)

	httpServer := server.Start(e, Logger, cfg.HTTPServerPort)

	if err := server.Stop(httpServer, Logger); err != nil {
		Logger.Error(ctx, "Error during server shutdown: "+err.Error())
	}
}
