package config

import (
	"fmt"
	"tic_tac_toe/pkg/db/redis"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTPServerPort int `env:"HTTP_SERVER_PORT" env-default:"8080"`
	redis.ConfigRedis
}

func New() *Config {
	cfg := Config{}
	err := cleanenv.ReadConfig("conf/local.env", &cfg)
	if err != nil {
		fmt.Printf("Schematic reading error: %v\n", err)
		return nil
	}

	fmt.Printf("Config loaded: %+v\n", cfg)
	return &cfg
}
