package utils

import (
	"os"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

func ParseEnv[T any](path string) (T, error) {
	var zero T
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load(path)
		if err != nil {
			return zero, err
		}
	}

	var cfg T
	err := env.Parse(&cfg)
	if err != nil {
		return zero, err
	}

	return cfg, nil
}
