package env

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Region    string `env:"PHAROS_REGION"`
	DebugHTTP bool   `env:"PHAROS_DEBUG_HTTP"`
}

func LoadConfig(ctx context.Context) (*Config, error) {
	config := Config{}

	if err := godotenv.Load(".env.local"); err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
	}

	if err := envconfig.Process(ctx, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
