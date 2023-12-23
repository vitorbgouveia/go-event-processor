package aws

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
)

func (s *sqsClientMock) SendMessage(*sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	return &sqs.SendMessageOutput{}, nil
}

func (s *sqsClientMock) GetQueueUrl(*sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	return &sqs.GetQueueUrlOutput{}, nil
}

func (s *sqsClientMockSendMessageErr) SendMessage(*sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
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

func TestMessageBroker_SendMessage_config_Err(t *testing.T) {
	t.Parallel()

	m := NewMessageBroker(session.Must(session.NewSessionWithOptions(
		session.Options{
			SharedConfigState: session.SharedConfigEnable,
		},
	)))

	err := m.SendMessage(SendMessageInput{
		QueueName: "___event_queue___", MessageID: uuid.NewString(), MessageBody: `{"content": "success_payment"}`,
	})
	assert.ErrorIs(t, err, aws.ErrMissingRegion)
}

func TestMessageBrotker_SendMessage_GetQueueURLErr(t *testing.T) {
	m := &messagebroker{
		client: &sqsClientMockGetQueueErr{},
	}

	err := m.SendMessage(SendMessageInput{
		QueueName:   "event_queue",
		MessageID:   uuid.NewString(),
		MessageBody: `{"content": 2000}`,
	})
	assert.EqualError(t, err, sqs.ErrCodeQueueDeletedRecently)
}

func TestMessageBrotker_SendMessage_Err(t *testing.T) {
	m := &messagebroker{
		client: &sqsClientMockSendMessageErr{},
	}

	err := m.SendMessage(SendMessageInput{
		QueueName:   "event_queue",
		MessageID:   uuid.NewString(),
		MessageBody: `{"content": 2000}`,
	})
	assert.EqualError(t, err, sqs.ErrCodeQueueDoesNotExist)
}

func TestMessageBroker_SendMessage(t *testing.T) {
	m := &messagebroker{
		client: &sqsClientMock{},
	}
	validUUID := uuid.NewString()
	validStringJson := `{"content": "event_1"}`

	testCase := []struct {
		description string
		input       SendMessageInput
		err         error
	}{
		{
			description: "should return error when message body is invalid json",
			input: SendMessageInput{
				QueueName:   "event_queue",
				MessageID:   validUUID,
				MessageBody: "content",
			},
			err: is.ErrJSON,
		},
		{
			description: "should return error when queue name is empty",
			input: SendMessageInput{
				QueueName:   "",
				MessageID:   validUUID,
				MessageBody: validStringJson,
			},
			err: validation.ErrRequired,
		},
		{
			description: "should return error when messageID id invalid",
			input: SendMessageInput{
				QueueName:   "event_queue",
				MessageID:   "invalid",
				MessageBody: validStringJson,
			},
			err: is.ErrUUID,
		},
		{
			description: "should return success when structValue and table name is valid",
			input: SendMessageInput{
				QueueName:   "event_queue",
				MessageID:   validUUID,
				MessageBody: validStringJson,
			},
			err: nil,
		},
	}

	for i, tc := range testCase {
		t.Run(fmt.Sprintf("TestMessageBroker_SendMessage_%d_%s", i, tc.description), func(t *testing.T) {
			err := m.SendMessage(tc.input)

			if tc.err != nil {
				assert.ErrorContains(t, err, tc.err.Error())
				return
			}

			assert.Nil(t, err)
		})
	}
}
