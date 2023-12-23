package worker

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vitorbgouveia/go-event-processor/internal/models"
	"github.com/vitorbgouveia/go-event-processor/internal/repository"
	repomocks "github.com/vitorbgouveia/go-event-processor/internal/repository/mocks"
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
	awsmocks "github.com/vitorbgouveia/go-event-processor/pkg/aws/mocks"
)

func TestProcessEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMsgBrotker := awsmocks.NewMockMessageBroker(ctrl)
	mockMsgBrotker.EXPECT().SendMessage(aws.SendMessageInput{}).MaxTimes(0)

	messageID := uuid.NewString()
	tenant := uuid.NewString()
	eventId := uuid.NewString()

	mockRepo := repomocks.NewMockDispatchedEvents(ctrl)
	mockRepo.EXPECT().Insert(repository.EventInsertInput{
		EventId: eventId, Context: "monitoring", Type: "inbound", Tenant: tenant, Data: `{"payment": 2000}`,
	}).Return(nil)

	w := NewDispatchedEventProcessor(&DispatchedEventProcessorInput{
		MsgBroker: mockMsgBrotker, Repo: mockRepo, QueueRetryProcess: "queue_to_rety", QueueDLQProcess: "queue_to_dlq",
	})

	err := w.ProcessEvents(context.Background(), &models.DispatchedEvent{
		Records: []models.EventRecord{
			{
				MessageId: messageID,
				Body: fmt.Sprintf(`{"event_id": "%s", "context": "monitoring", "type": "inbound", "tenant": "%s", "data": "{\"payment\": 2000}"}`,
					eventId, tenant), EventARN: "any",
			},
		},
	}, make(chan struct{}))
	assert.Nil(t, err)
}

func TestProcessEvents_with_cancel_signal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	msgId := uuid.NewString()
	body := fmt.Sprintf(`{"context": "monitoring", "type": "inbound", "tenant": "%s", "data": "{\"payment\": 2000}"}`, msgId)
	retryErr := errors.New("could not sent message to retry")

	mockMsgBrotker := awsmocks.NewMockMessageBroker(ctrl)
	gomock.InOrder(
		mockMsgBrotker.EXPECT().SendMessage(aws.SendMessageInput{
			QueueName: "retry_queue", MessageID: msgId, MessageBody: body,
		}).Return(nil),
		mockMsgBrotker.EXPECT().SendMessage(aws.SendMessageInput{
			QueueName: "retry_queue", MessageID: msgId, MessageBody: body,
		}).Return(retryErr),
	)

	mockRepo := repomocks.NewMockDispatchedEvents(ctrl)
	mockRepo.EXPECT().Insert(repository.EventInsertInput{}).MaxTimes(0)

	w := NewDispatchedEventProcessor(&DispatchedEventProcessorInput{
		MsgBroker: mockMsgBrotker, Repo: mockRepo, QueueRetryProcess: "retry_queue", QueueDLQProcess: "queue_to_dlq",
	})

	cancel := make(chan struct{})
	close(cancel)
	err := w.ProcessEvents(context.Background(), &models.DispatchedEvent{
		Records: []models.EventRecord{{MessageId: msgId, Body: body, EventARN: "any"}},
	}, cancel)
	assert.Nil(t, err)

	err = w.ProcessEvents(context.Background(), &models.DispatchedEvent{
		Records: []models.EventRecord{{MessageId: msgId, Body: body, EventARN: "any"}},
	}, cancel)
	assert.ErrorIs(t, err, retryErr)
}

func TestProcessEvents_message_is_requeue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	msgId := uuid.NewString()
	tenant := uuid.NewString()
	bodyWithoutType := fmt.Sprintf(`{"context": "monitoring", "tenant": "%s", "data": "{\"payment\": 2000}"}`, tenant)

	mockMsgBrotker := awsmocks.NewMockMessageBroker(ctrl)
	mockMsgBrotker.EXPECT().SendMessage(aws.SendMessageInput{
		QueueName: "retry_queue", MessageID: msgId, MessageBody: bodyWithoutType,
	}).MaxTimes(0)

	mockRepo := repomocks.NewMockDispatchedEvents(ctrl)
	mockRepo.EXPECT().Insert(repository.EventInsertInput{}).MaxTimes(0)

	w := NewDispatchedEventProcessor(&DispatchedEventProcessorInput{
		MsgBroker: mockMsgBrotker, Repo: mockRepo, QueueRetryProcess: "retry_queue", QueueDLQProcess: "queue_to_dlq",
	})

	err := w.ProcessEvents(context.Background(), &models.DispatchedEvent{
		Records: []models.EventRecord{{MessageId: msgId, Body: bodyWithoutType, EventARN: "retry_queue"}},
	}, make(<-chan struct{}))
	assert.ErrorIs(t, err, ErrEventSentToRetryAgain)
}

func TestProcessEvents_message_body(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	msgId := uuid.NewString()
	tenant := uuid.NewString()
	eventId := uuid.NewString()
	eventARN := "event_arn"
	validBody := fmt.Sprintf(`{"event_id": "%s", "context": "monitoring", "type": "inbound", "tenant": "%s", "data": "{\"payment\": 2000}"}`,
		eventId, tenant)
	bodyWithoutContext := fmt.Sprintf(`{"type": "inbound", "tenant": "%s", "data": "{\"payment\": 2000}"}`, tenant)
	insertInput := repository.EventInsertInput{
		EventId: eventId, Context: "monitoring", Type: "inbound", Tenant: tenant, Data: `{"payment": 2000}`,
	}
	retryErr := errors.New("could not sent message to retry")

	mockMsgBrotker := awsmocks.NewMockMessageBroker(ctrl)
	gomock.InOrder(
		mockMsgBrotker.EXPECT().SendMessage(aws.SendMessageInput{
			QueueName: "retry_queue", MessageID: msgId, MessageBody: bodyWithoutContext,
		}).Return(nil),
		mockMsgBrotker.EXPECT().SendMessage(aws.SendMessageInput{
			QueueName: "retry_queue", MessageID: msgId, MessageBody: validBody,
		}).Return(nil),
		mockMsgBrotker.EXPECT().SendMessage(aws.SendMessageInput{
			QueueName: "retry_queue", MessageID: msgId, MessageBody: bodyWithoutContext,
		}).Return(retryErr),
		mockMsgBrotker.EXPECT().SendMessage(aws.SendMessageInput{
			QueueName: "retry_queue", MessageID: msgId, MessageBody: validBody,
		}).Return(retryErr),
		mockMsgBrotker.EXPECT().SendMessage(aws.SendMessageInput{
			QueueName: "retry_queue", MessageID: msgId, MessageBody: "ok",
		}).Return(nil),
	)

	mockRepo := repomocks.NewMockDispatchedEvents(ctrl)
	gomock.InOrder(
		mockRepo.EXPECT().Insert(insertInput).Return(nil),
		mockRepo.EXPECT().Insert(insertInput).Return(aws.ErrInvalidInsertInput),
		mockRepo.EXPECT().Insert(insertInput).Return(aws.ErrInvalidInsertInput),
	)

	w := NewDispatchedEventProcessor(&DispatchedEventProcessorInput{
		MsgBroker: mockMsgBrotker, Repo: mockRepo, QueueRetryProcess: "retry_queue", QueueDLQProcess: "queue_to_dlq",
	})

	testCase := []struct {
		description string
		eventRecord models.EventRecord
		err         error
	}{
		{
			description: "should pass when body is valid",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      validBody,
			},
			err: nil,
		},
		{
			description: "should send to retry err when body is invalid",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      bodyWithoutContext,
			},
			err: nil,
		},
		{
			description: "should send to retry when insert return error",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      validBody,
			},
			err: nil,
		},
		{
			description: "should return err when body is invalid and retry fail",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      bodyWithoutContext,
			},
			err: retryErr,
		},
		{
			description: "should return err when insert fail and retry fail",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      validBody,
			},
			err: retryErr,
		},
		{
			description: "should send to retry when body is not json",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      "ok",
			},
			err: nil,
		},
	}
	for i, tc := range testCase {
		t.Run(fmt.Sprintf("TestProcessEvents_%d_%s", i, tc.description), func(t *testing.T) {
			err := w.ProcessEvents(context.Background(), &models.DispatchedEvent{
				Records: []models.EventRecord{
					tc.eventRecord,
				},
			}, make(<-chan struct{}))

			if tc.err != nil {
				assert.ErrorContains(t, err, tc.err.Error())
				return
			}

			assert.Nil(t, err)
		})
	}
}
