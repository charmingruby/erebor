package env

import (
	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

// Load attempts to load a .env file, then parses environment variables into T.
func Load[T any]() (*T, error) {
	_ = godotenv.Load()

	var cfg T
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
