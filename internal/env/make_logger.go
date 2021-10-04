package env

import (
	zap "go.uber.org/zap"
)

func MakeLogger() (*zap.Logger, error) {
	logConfig := zap.NewProductionConfig()
	logConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	logConfig.Encoding = "json"

	return logConfig.Build()
}
