package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	MsgChannelBufSize int    `env:"MSG_CHANNEL_BUFFER_SIZE" envDefault:"100"`
	GRPCPort          string `env:"GRPC_PORT" envDefault:":50051"`
	LogLevel          string `env:"LOG_LEVEL" envDefault:"info"`
}

func InitServiceConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		// Ignore missing .env file, use environment variables instead
	}
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
