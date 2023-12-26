package main

import (
	"flag"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/vitorbgouveia/go-event-processor/internal/repository"
	"github.com/vitorbgouveia/go-event-processor/pkg"
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
)

var (
	queueRetryProcess  string
	validEventTopic    string
	rejectedEventTopic string
	region             string
	endpoint           string
)

func init() {
	flag.StringVar(&queueRetryProcess, "event_processor_retry_queue", "event_processor_retry", "send message to retry queue when has error")
	flag.StringVar(&validEventTopic, "valid_event_topic", "valid_event", "publish event when event received is valid")
	flag.StringVar(&rejectedEventTopic, "reject_event_topic", "rejected_event", "publish event when event received not is valid")
	flag.StringVar(&region, "region", "us-east-1", "region of cloud services")
	flag.StringVar(&endpoint, "endpoint", "http://localstack:4566", "endpoint of cloud services")
	flag.Parse()
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
		panic(fmt.Sprintf("could not load aws config: %v", err))
	}

	msgBrotker := aws.NewMessageBroker(sess)
	if err := msgBrotker.CreateRejectEventTopic(rejectedEventTopic); err != nil {
		panic(fmt.Sprintf("could not start message broker: %v", err))
	}
	if err := msgBrotker.CreateValidEventTopic(validEventTopic); err != nil {
		panic(fmt.Sprintf("could not start message broker: %v", err))
	}

	persistence := aws.NewPersistence(sess)
	repo := repository.NewValidEvents(persistence)

	handler := pkg.NewLambdaHandler(pkg.LambdaHandlerInput{
		QueueRetryProcess: queueRetryProcess, MsgBroker: msgBrotker, DispatchedEventRepo: repo,
	})

	lambda.Start(handler.Handle)
}
