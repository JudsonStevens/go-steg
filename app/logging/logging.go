package logging

import (
	"go.uber.org/zap"
)

func NewLogger(logFile string) *zap.SugaredLogger {
	zap.NewProductionConfig()
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	if logFile != "" {
		config.OutputPaths = append(config.OutputPaths, logFile)
	}
	logger, err := config.Build(zap.AddCaller())
	if err != nil {
		panic(err)
	}
	return logger.Sugar()
}
