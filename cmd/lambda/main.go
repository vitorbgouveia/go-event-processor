package main

import (
	"flag"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/vitorbgouveia/go-event-processor/internal/repository"
	"github.com/vitorbgouveia/go-event-processor/pkg"
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
)

var (
	queueRetryProcess string
	queueDLQProcess   string
	region            string
	endpoint          string
)

func init() {
	flag.StringVar(&queueRetryProcess, "retry_queue_name", "dispatch_event_processor_retry", "send message to queue when has error")
	flag.StringVar(&queueDLQProcess, "dlq_queue_name", "dispatch_event_processor_DLQ", "send message to queue when have an incompatible contract")
	flag.StringVar(&region, "region", "us-east-1", "region of cloud services")
	flag.StringVar(&endpoint, "endpoint", "http://localstack:4566", "endpoint of cloud services")
}

// aws sqs send-message --endpoint-url=http://localhost:4566 --queue-url http://localhost:4576/000000000000/dispatch_event_processor --region us-east-1 --message-body '{Test Message!}'
/*
 aws sqs send-message --endpoint-url=http://localhost:4566 --queue-url http://localhost:4576/000000000000/dispatch_event_processor --region us-east-1 \
 --message-body '{"event_id": "939f9390-db94-4529-8069-4383d087bd07", "context": "ok", "type": "ok", "tenant": "2e7d5843-a6db-4909-8add-79258783a524", "data": "{\"ok\": \"test\"}"}'
*/

func main() {
	sess, err := session.NewSession(&awssdk.Config{
		Region:   &region,
		Endpoint: &endpoint,
	})
	if err != nil {
		log.Fatalln("could not load aws config", err)
		return
	}

	msgBrotker := aws.NewMessageBroker(sess)
	persistence := aws.NewPersistence(sess)

	repo := repository.NewDispatchedEvents(persistence)

	handler := pkg.NewLambdaHandler(pkg.LambdaHandlerInput{
		QueueRetryProcess: queueRetryProcess, QueueDLQProcess: queueDLQProcess, MsgBroker: msgBrotker, DispatchedEventRepo: repo,
	})

	lambda.Start(handler.Handle)
}
