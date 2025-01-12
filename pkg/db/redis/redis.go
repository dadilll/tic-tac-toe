package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type ConfigRedis struct {
	Addr     string `env:"REDIS_HOST" env-default:"6379"`
	Password string `env:"REDIS_PASSWORD" env-default:""`
	DB       int    `env:"REDIS_DB" env-default:"0"`
}

func New(config ConfigRedis) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Addr,     // Адрес Redis
		Password: config.Password, // Пароль Redis (если есть)
		DB:       config.DB,       // Номер базы данных
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return rdb, nil
}
