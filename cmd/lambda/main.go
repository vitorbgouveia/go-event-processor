package main

import (
	"context"
	"flag"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/vitorbgouveia/go-event-processor/internal/repository"
	"github.com/vitorbgouveia/go-event-processor/pkg"
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
)

var (
	queueRetryProcess string
	queueDLQProcess   string
)

func init() {
	flag.StringVar(&queueRetryProcess, "retry_queue_name", "dispatch_event_processor_retry", "send message to queue when has error")
	flag.StringVar(&queueDLQProcess, "dlq_queue_name", "dispatch_event_processor_DLQ", "send message to queue when have an incompatible contract")
}

// aws sqs send-message --endpoint-url=http://localhost:4566 --queue-url http://localhost:4576/000000000000/dispatch_event_processor --region us-east-1 --message-body '{Test Message!}'
// aws sqs send-message --endpoint-url=http://localhost:4566 --queue-url http://localhost:4576/000000000000/dispatch_event_processor --region us-east-1 --message-body '{"context": "ok", "type": "ok", "tenant": "ok", "event_data": "ok"}'

func main() {
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalln("could not load aws config", err)
		return
	}

	msgBrotker := aws.NewMessageBroker(awsCfg)
	persistence := aws.NewPersistence(awsCfg)

	repo := repository.NewDispatchedEvents(ctx, persistence)

	handler := pkg.NewLambdaHandler(pkg.LambdaHandlerInput{
		QueueRetryProcess: queueRetryProcess, QueueDLQProcess: queueDLQProcess, MsgBroker: msgBrotker, DispatchedEventRepo: repo,
	})

	lambda.Start(handler.Handle)
}
