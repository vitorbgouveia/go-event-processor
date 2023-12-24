package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/stretchr/testify/assert"
)

var (
	errInvalidTopicName = errors.New("invalid topic name")
	errConnection       = errors.New("connection error")
)

type (
	sqsClientMock struct {
		sqsiface.SQSAPI
	}
	sqsClientMockGetQueueErr struct {
		sqsiface.SQSAPI
	}
	sqsClientMockSendMessageErr struct {
		sqsiface.SQSAPI
	}
	snsClientMock struct {
		snsiface.SNSAPI
	}
	snsClientMockErr struct {
		snsiface.SNSAPI
	}
)

func (s *sqsClientMock) SendMessageWithContext(aws.Context, *sqs.SendMessageInput, ...request.Option) (*sqs.SendMessageOutput, error) {
	return &sqs.SendMessageOutput{}, nil
}

func (s *sqsClientMock) GetQueueUrl(*sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	return &sqs.GetQueueUrlOutput{}, nil
}

func (s *sqsClientMockSendMessageErr) SendMessageWithContext(aws.Context, *sqs.SendMessageInput, ...request.Option) (*sqs.SendMessageOutput, error) {
	return nil, errors.New(sqs.ErrCodeQueueDoesNotExist)
}

func (s *sqsClientMockSendMessageErr) GetQueueUrl(*sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	return &sqs.GetQueueUrlOutput{
		QueueUrl: aws.String("url_queue"),
	}, nil
}

func (s *sqsClientMockGetQueueErr) GetQueueUrl(*sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	return nil, errors.New(sqs.ErrCodeQueueDeletedRecently)
}

func (s *snsClientMock) CreateTopic(i *sns.CreateTopicInput) (*sns.CreateTopicOutput, error) {
	return &sns.CreateTopicOutput{TopicArn: aws.String(*i.Name)}, nil
}

func (s *snsClientMock) PublishWithContext(aws.Context, *sns.PublishInput, ...request.Option) (*sns.PublishOutput, error) {
	return nil, nil
}

func (s *snsClientMockErr) CreateTopic(i *sns.CreateTopicInput) (*sns.CreateTopicOutput, error) {
	return nil, errInvalidTopicName
}

func (s *snsClientMockErr) PublishWithContext(aws.Context, *sns.PublishInput, ...request.Option) (*sns.PublishOutput, error) {
	return nil, errConnection
}

func TestMessageBroker_SendMessage_config_Err(t *testing.T) {
	t.Parallel()

	m := NewMessageBroker(session.Must(session.NewSessionWithOptions(
		session.Options{
			SharedConfigState: session.SharedConfigEnable,
		},
	)))
	assert.IsType(t, m, &messagebroker{})
}

func TestMessageBroker_CreateValidEventTopic(t *testing.T) {
	m := &messagebroker{
		snsClient: &snsClientMockErr{},
	}
	err := m.CreateValidEventTopic("valid_event")
	assert.EqualError(t, err, errInvalidTopicName.Error())
	assert.False(t, strings.Contains(m.validEventTopicARN, "valid_event"))

	m.snsClient = &snsClientMock{}
	err = m.CreateValidEventTopic("valid_event")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(m.validEventTopicARN, "valid_event"))
}

func TestMessageBroker_CreateRejectEventTopic(t *testing.T) {
	m := &messagebroker{
		snsClient: &snsClientMockErr{},
	}
	err := m.CreateRejectEventTopic("rejected_event")
	assert.EqualError(t, err, errInvalidTopicName.Error())
	assert.False(t, strings.Contains(m.rejectedEventTopicARN, "rejected_event"))

	m.snsClient = &snsClientMock{}
	err = m.CreateRejectEventTopic("rejected_event")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(m.rejectedEventTopicARN, "rejected_event"))
}

func TestMessageBroker_PublishValidatedEvent(t *testing.T) {
	ctx := context.Background()
	m := &messagebroker{
		snsClient: &snsClientMock{},
	}
	err := m.PublishValidEvent(ctx, "content_message")
	assert.Nil(t, err)

	m.snsClient = &snsClientMockErr{}
	err = m.PublishValidEvent(ctx, "content_message")
	assert.EqualError(t, err, errConnection.Error())
}

func TestMessageBroker_PublishRejectedEvent(t *testing.T) {
	ctx := context.Background()
	m := &messagebroker{
		snsClient: &snsClientMock{},
	}
	err := m.PublishRejectedEvent(ctx, "content_message", errors.New("invalid body"))
	assert.Nil(t, err)

	m.snsClient = &snsClientMockErr{}
	err = m.PublishRejectedEvent(ctx, "content_message", errors.New("invalid body"))
	assert.EqualError(t, err, errConnection.Error())
}

func TestMessageBrotker_SendMessage_GetQueueURLErr(t *testing.T) {
	m := &messagebroker{
		sqsClient: &sqsClientMockGetQueueErr{},
	}

	err := m.SendToQueue(context.Background(), SendToQueueInput{
		QueueName:   "event_queue",
		MessageBody: `{"content": 2000}`,
	})
	assert.EqualError(t, err, sqs.ErrCodeQueueDeletedRecently)
}

func TestMessageBrotker_SendMessage_Err(t *testing.T) {
	m := &messagebroker{
		sqsClient: &sqsClientMockSendMessageErr{},
	}

	err := m.SendToQueue(context.Background(), SendToQueueInput{
		QueueName:   "event_queue",
		MessageBody: `{"content": 2000}`,
	})
	assert.EqualError(t, err, sqs.ErrCodeQueueDoesNotExist)
}

func TestMessageBroker_SendMessage(t *testing.T) {
	m := &messagebroker{
		sqsClient: &sqsClientMock{},
	}
	validStringJson := `{"content": "event_1"}`
	ctx := context.Background()

	testCase := []struct {
		description string
		input       SendToQueueInput
		err         error
	}{
		{
			description: "should return error when message body is invalid json",
			input: SendToQueueInput{
				QueueName:   "event_queue",
				MessageBody: "content",
			},
			err: is.ErrJSON,
		},
		{
			description: "should return error when queue name is empty",
			input: SendToQueueInput{
				QueueName:   "",
				MessageBody: validStringJson,
			},
			err: validation.ErrRequired,
		},
		{
			description: "should return success when structValue and table name is valid",
			input: SendToQueueInput{
				QueueName:   "event_queue",
				MessageBody: validStringJson,
			},
			err: nil,
		},
	}

	for i, tc := range testCase {
		t.Run(fmt.Sprintf("TestMessageBroker_SendMessage_%d_%s", i, tc.description), func(t *testing.T) {
			err := m.SendToQueue(ctx, tc.input)

			if tc.err != nil {
				assert.ErrorContains(t, err, tc.err.Error())
				return
			}

			assert.Nil(t, err)
		})
	}
}
