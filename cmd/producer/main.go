package main

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

var (
	eventProcessorQueue string
	rejectedEventTopic  string
	region              string
	endpoint            string
	awsAccessKey        string
	awsSecret           string
)

type sendEventInput struct {
	sqsClient sqsiface.SQSAPI
	queueUrl  string
	msg       string
}

func init() {
	flag.StringVar(&eventProcessorQueue, "event_processor_queue", "event_processor", "send message to process")
	flag.StringVar(&region, "region", "us-east-1", "region of cloud services")
	flag.StringVar(&endpoint, "endpoint", "http://localhost:4566", "endpoint of cloud services")
	flag.StringVar(&awsAccessKey, "aws_access_key", "cred_key_id", "aws accessKey of cloud services")
	flag.StringVar(&awsSecret, "aws_secret_key", "cred_secret_key", "aws secretKey cloud services")
}

const (
	validMessage    = `{"event_id": "80602061-a0e8-42e1-9331-efbb7c408abf", "context": "monitoring", "type": "inbound", "tenant": "9ccc4d9d-1fa8-4b84-bb25-c187e1269876", "data": "{\"payment\": 2000}"}`
	invalidMessage  = `{"context": "monitoring", "type": "inbound", "tenant": "9ccc4d9d-1fa8-4b84-bb25-c187e1269876", "data": "{\"payment\": 2000}"}`
	semaphoreLength = 1
)

func main() {
	sess, err := session.NewSession(&aws.Config{
		Endpoint: &endpoint,
		Region:   &region,
		Credentials: credentials.NewStaticCredentials(
			"cred_key_id", "cred_secret_key", ""),
	})
	if err != nil {
		panic(fmt.Sprintf("could not load aws config: %v", err))
	}

	sqsClient := sqs.New(sess)

	urlOutput, err := sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(eventProcessorQueue),
	})
	if err != nil {
		panic(fmt.Sprintf("could not get url queue: %v", err))
	}

	errChan := make(chan error)
	wg := sync.WaitGroup{}
	for i := 0; i < semaphoreLength*1; i++ {
		wg.Add(1)
		// go func(errChan chan<- error) {
		// 	defer wg.Done()
		// 	if err := sendEvents(sendEventInput{
		// 		sqsClient: sqsClient,
		// 		queueUrl:  *urlOutput.QueueUrl,
		// 		msg:       validMessage,
		// 	}); err != nil {
		// 		errChan <- err
		// 	}
		// 	time.Sleep(time.Millisecond * 20)
		// }(errChan)

		go func(errChan chan<- error) {
			defer wg.Done()
			if err := sendEvents(sendEventInput{
				sqsClient: sqsClient,
				queueUrl:  *urlOutput.QueueUrl,
				msg:       invalidMessage,
			}); err != nil {
				errChan <- err
			}
			time.Sleep(time.Millisecond * 20)
		}(errChan)
	}

	finishChan := make(chan error)

	go func(finishChan chan<- error) {
		wg.Wait()
		finishChan <- nil
	}(finishChan)

	go func(finishChan chan<- error) {
		err, ok := <-errChan
		if ok {
			finishChan <- err
		}
	}(finishChan)

	err = <-finishChan
	if err != nil {
		panic(fmt.Sprintf("could not process the events, %v", err))
	}
}

func sendEvents(i sendEventInput) error {
	_, err := i.sqsClient.SendMessageWithContext(context.Background(), &sqs.SendMessageInput{
		MessageBody: &i.msg,
		QueueUrl:    &i.queueUrl,
	})
	return err
}
