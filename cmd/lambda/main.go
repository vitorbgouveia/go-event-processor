package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	"github.com/aws/aws-lambda-go/lambda"

	"go.uber.org/zap"

	"github.com/vitorbgouveia/go-event-processor/pkg"
	"github.com/vitorbgouveia/go-event-processor/pkg/messagebroker"
	"github.com/vitorbgouveia/go-event-processor/pkg/services/worker"
)

var (
	logger            *zap.SugaredLogger
	queueRetryProcess string
	queueDLQProcess   string
)

func init() {
	flag.StringVar(&queueRetryProcess, "retry_queue_name", "dispatch_event_processor_retry", "send message to queue when has error")
	flag.StringVar(&queueDLQProcess, "dlq_queue_name", "dispatch_event_processor_DLQ", "send message to queue when have an incompatible contract")
}

func HandleRequest(ctx context.Context, data interface{}) (*string, error) {
	logger = pkg.NewLogger()
	defer logger.Sync()

	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	msgBrotker, err := messagebroker.New(logger, ctx)
	if err != nil {
		return nil, err
	}

	w := worker.NewDispatchEventProcessor(&worker.DispatchEventProcess{
		Logger: logger, MsgBroker: msgBrotker, QueueRetryProcess: queueRetryProcess, QueueDLQProcess: queueDLQProcess,
	})
	if err := w.ProcessEvents(ctx, data, sigCtx.Done()); err != nil {
		return nil, err
	}

	ok := "ok"
	return &ok, nil
}

// aws sqs send-message --endpoint-url=http://localhost:4566 --queue-url http://localhost:4576/000000000000/dispatch_event_processor --region us-east-1 --message-body '{Test Message!}'

func main() {
	lambda.Start(HandleRequest)
}
