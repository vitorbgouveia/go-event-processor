package pkg

import (
	"os"

	"go.uber.org/zap"
)

const (
	MessageIDKey = "message_id"
	QueueNameKey = "queue_name"
	QueuUrlKey   = "queue_url"
	EventBodyKey = "event_body"
)

func NewLogger() *zap.SugaredLogger {
	var initLogger *zap.Logger
	initLogger, _ = zap.NewProduction()

	if os.Getenv("IS_DEV") == "1" {
		initLogger, _ = zap.NewDevelopment()
	}

	return initLogger.Sugar()
}
