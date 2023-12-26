package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
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
	qtdEventsSend       int
	sendEventSemaphore  int
)

type sendEventInput struct {
	sqsClient sqsiface.SQSAPI
	queueUrl  string
	msg       string
}

func init() {
	flag.StringVar(&eventProcessorQueue, "event-processor-queue", "event_processor", "send message to process")
	flag.StringVar(&region, "region", "us-east-1", "region of cloud services")
	flag.StringVar(&endpoint, "endpoint", "http://localhost:4566", "endpoint of cloud services")
	flag.StringVar(&awsAccessKey, "aws-access-key", "cred_key_id", "aws accessKey of cloud services")
	flag.StringVar(&awsSecret, "aws-secret-key", "cred_secret_key", "aws secretKey cloud services")
	flag.IntVar(&qtdEventsSend, "qtd-events-send", 1, "quantity of events sent to lambda")
	flag.IntVar(&sendEventSemaphore, "event-send-semaphore", 50, "limit of events sent in parallel to lambda")
	flag.Parse()
}

const (
	validMessage   = `{"event_id": "80602061-a0e8-42e1-9331-efbb7c408abf", "context": "monitoring", "type": "inbound", "tenant": "9ccc4d9d-1fa8-4b84-bb25-c187e1269876", "data": "{\"payment\": 2000}"}`
	invalidMessage = `{"context": "monitoring", "type": "inbound", "tenant": "9ccc4d9d-1fa8-4b84-bb25-c187e1269876", "data": "{\"payment\": 2000}"}`
)

func main() {
	ctx := context.Background()
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sess, err := session.NewSession(&aws.Config{
		Endpoint: &endpoint,
		Region:   &region,
		Credentials: credentials.NewStaticCredentials(
			awsAccessKey, awsSecret, ""),
	})
	if err != nil {
		panic(fmt.Sprintf("could not load aws config: %v", err))
	}

	snsClient := sns.New(sess)
	sqsClient := sqs.New(sess)
	urlOutput, err := sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(eventProcessorQueue),
	})
	if err != nil {
		panic(fmt.Sprintf("could not get url queue: %v", err))
	}
	cancelListenRejectedEvents := make(chan bool)
	defer close(cancelListenRejectedEvents)
	listenRejectedEvents(sqsClient, snsClient, cancelListenRejectedEvents)

	errChan := make(chan error)
	wg := sync.WaitGroup{}
	for i := 0; i < qtdEventsSend; i++ {
		wg.Add(2)
		go func(errChan chan<- error) {
			defer wg.Done()
			if err := sendEvents(sendEventInput{
				sqsClient: sqsClient,
				queueUrl:  *urlOutput.QueueUrl,
				msg:       validMessage,
			}); err != nil {
				errChan <- err
			}
		}(errChan)

		go func(errChan chan<- error) {
			defer wg.Done()
			if err := sendEvents(sendEventInput{
				sqsClient: sqsClient,
				queueUrl:  *urlOutput.QueueUrl,
				msg:       invalidMessage,
			}); err != nil {
				errChan <- err
			}
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
		panic(fmt.Sprintf("fail to send events, %v", err))
	}

	<-sigCtx.Done()
	cancelListenRejectedEvents <- true
}

func sendEvents(i sendEventInput) error {
	_, err := i.sqsClient.SendMessageWithContext(context.Background(), &sqs.SendMessageInput{
		MessageBody: &i.msg,
		QueueUrl:    &i.queueUrl,
	})
	return err
}

func listenRejectedEvents(sqsClient sqsiface.SQSAPI, snsClient snsiface.SNSAPI, cancel <-chan bool) error {
	queueO, err := sqsClient.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String("rejected_event_processor"),
	})
	if err != nil {
		return err
	}

	topicO, err := snsClient.CreateTopic(&sns.CreateTopicInput{
		Name: aws.String("rejected_event"),
	})
	if err != nil {
		return err
	}

	queueAttr, err := sqsClient.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       queueO.QueueUrl,
		AttributeNames: []*string{aws.String("QueueArn")},
	})
	if err != nil {
		return err
	}
	qARN := queueAttr.Attributes["QueueArn"]

	snsClient.Subscribe(&sns.SubscribeInput{
		Endpoint: qARN,
		Protocol: aws.String("sqs"),
		TopicArn: topicO.TopicArn,
	})

	go func() {
		rejectedMessage := 1
		for {
			select {
			case <-cancel:
				slog.Info("listen canceled")
				return
			default:
				receiveO, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
					QueueUrl: queueO.QueueUrl,
				})
				if err != nil {
					slog.Error("could not receive message of sqsClient", slog.Any("err", err))
					continue
				}

				for _, msg := range receiveO.Messages {
					slog.Info(fmt.Sprintf("%d-received rejected event notification", rejectedMessage))
					rejectedMessage += 1
					if _, err := sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
						QueueUrl:      queueO.QueueUrl,
						ReceiptHandle: msg.ReceiptHandle,
					}); err != nil {
						slog.Error("could not delete message", slog.Any("err", err))
					}
				}

				time.Sleep(time.Second * 2)
			}
		}
	}()

	return nil
}
