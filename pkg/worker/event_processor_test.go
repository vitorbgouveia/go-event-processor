package worker

import (
	"context"
	"encoding/json"
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

	tenant := uuid.NewString()
	eventId := uuid.NewString()
	validBody := fmt.Sprintf(`{"event_id": "%s", "context": "monitoring", "type": "inbound", "tenant": "%s", "data": "{\"payment\": 2000}"}`,
		eventId, tenant)

	mockMsgBroker := awsmocks.NewMockMessageBroker(ctrl)
	mockMsgBroker.EXPECT().SendToQueue(context.Background(), aws.SendToQueueInput{}).MaxTimes(0)
	mockMsgBroker.EXPECT().PublishValidEvent(context.Background(), validBody).MaxTimes(1).Return(nil)

	messageID := uuid.NewString()

	mockRepo := repomocks.NewMockEvents(ctrl)
	mockRepo.EXPECT().Insert(repository.EventInsertInput{
		EventId: eventId, Context: "monitoring", Type: "inbound", Tenant: tenant, Data: `{"payment": 2000}`,
	}).Return(nil)

	w := NewDispatchedEventProcessor(&EventProcessorInput{
		MsgBroker: mockMsgBroker, Repo: mockRepo, QueueRetryProcess: "queue_to_rety",
	})

	err := w.ProcessEvents(context.Background(), &models.EventInput{
		Records: []models.EventRecord{
			{
				MessageId: messageID,
				Body:      validBody, EventARN: "any",
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
	ctx := context.Background()

	mockMsgBroker := awsmocks.NewMockMessageBroker(ctrl)
	gomock.InOrder(
		mockMsgBroker.EXPECT().SendToQueue(ctx, aws.SendToQueueInput{
			QueueName: "retry_queue", MessageBody: body,
		}).Return(nil),
		mockMsgBroker.EXPECT().SendToQueue(ctx, aws.SendToQueueInput{
			QueueName: "retry_queue", MessageBody: body,
		}).Return(retryErr),
	)

	mockRepo := repomocks.NewMockEvents(ctrl)
	mockRepo.EXPECT().Insert(repository.EventInsertInput{}).MaxTimes(0)

	w := NewDispatchedEventProcessor(&EventProcessorInput{
		MsgBroker: mockMsgBroker, Repo: mockRepo, QueueRetryProcess: "retry_queue",
	})

	cancel := make(chan struct{})
	close(cancel)
	err := w.ProcessEvents(context.Background(), &models.EventInput{
		Records: []models.EventRecord{{MessageId: msgId, Body: body, EventARN: "any"}},
	}, cancel)
	assert.Nil(t, err)

	err = w.ProcessEvents(context.Background(), &models.EventInput{
		Records: []models.EventRecord{{MessageId: msgId, Body: body, EventARN: "any"}},
	}, cancel)
	assert.ErrorIs(t, err, retryErr)
}

func TestProcessEvents_message_is_requeue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	msgId := uuid.NewString()
	tenant := uuid.NewString()
	eventId := uuid.NewString()
	validBody := fmt.Sprintf(
		`{"event_id": "%s", "type": "inbound", "context": "monitoring", "tenant": "%s", "data": "{\"payment\": 2000}"}`, eventId, tenant)
	errInsert := errors.New("fail to insert new item")

	mockMsgBrotker := awsmocks.NewMockMessageBroker(ctrl)
	mockMsgBrotker.EXPECT().SendToQueue(context.Background(), aws.SendToQueueInput{
		QueueName: "retry_queue", MessageBody: validBody,
	}).MaxTimes(1).Return(ErrEventSentToRetryAgain)

	mockRepo := repomocks.NewMockEvents(ctrl)
	mockRepo.EXPECT().Insert(repository.EventInsertInput{
		EventId: eventId, Type: "inbound", Context: "monitoring", Tenant: tenant, Data: `{"payment": 2000}`,
	}).MaxTimes(1).Return(errInsert)

	w := NewDispatchedEventProcessor(&EventProcessorInput{
		MsgBroker: mockMsgBrotker, Repo: mockRepo, QueueRetryProcess: "retry_queue",
	})

	err := w.ProcessEvents(context.Background(), &models.EventInput{
		Records: []models.EventRecord{{MessageId: msgId, Body: validBody, EventARN: "retry_queue"}},
	}, make(<-chan struct{}))
	assert.ErrorIs(t, err, ErrEventSentToRetryAgain)
}

func TestProcessEvents_message_body(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	msgId := uuid.NewString()
	tenant := uuid.NewString()
	eventId := uuid.NewString()
	ctx := context.Background()
	eventARN := "event_arn"
	validBody := fmt.Sprintf(`{"event_id": "%s", "context": "monitoring", "type": "inbound", "tenant": "%s", "data": "{\"payment\": 2000}"}`,
		eventId, tenant)
	bodyIncomplete := fmt.Sprintf(`{"type": "inbound", "tenant": "%s", "data": "{\"payment\": 2000}"}`, tenant)
	insertInput := repository.EventInsertInput{
		EventId: eventId, Context: "monitoring", Type: "inbound", Tenant: tenant, Data: `{"payment": 2000}`,
	}
	var messageBodyIncomplete models.EventMessageBody
	errIncompleteBody := json.Unmarshal([]byte(bodyIncomplete), &messageBodyIncomplete)
	assert.Nil(t, errIncompleteBody)
	retryErr := errors.New("could not sent message to retry")
	errSendEvent := errors.New("fail to send event")

	var body models.EventMessageBody
	errNoBody := json.Unmarshal([]byte("ok"), &body)

	mockMsgBrotker := awsmocks.NewMockMessageBroker(ctrl)
	gomock.InOrder(
		mockMsgBrotker.EXPECT().PublishValidEvent(ctx, validBody).Return(nil),
		mockMsgBrotker.EXPECT().PublishRejectedEvent(ctx, bodyIncomplete, messageBodyIncomplete.Validate()).Return(nil),
		mockMsgBrotker.EXPECT().SendToQueue(ctx, aws.SendToQueueInput{
			QueueName: "retry_queue", MessageBody: validBody,
		}).Return(nil),
		mockMsgBrotker.EXPECT().SendToQueue(ctx, aws.SendToQueueInput{
			QueueName: "retry_queue", MessageBody: validBody,
		}).Return(retryErr),
		mockMsgBrotker.EXPECT().PublishRejectedEvent(ctx, "ok", errNoBody).Return(nil),
		mockMsgBrotker.EXPECT().PublishValidEvent(ctx, validBody).Return(errSendEvent),
		mockMsgBrotker.EXPECT().SendToQueue(ctx, aws.SendToQueueInput{
			QueueName: "retry_queue", MessageBody: validBody,
		}).Return(retryErr),
		mockMsgBrotker.EXPECT().PublishRejectedEvent(ctx, bodyIncomplete, messageBodyIncomplete.Validate()).Return(errSendEvent),
		mockMsgBrotker.EXPECT().PublishValidEvent(ctx, validBody).Return(errSendEvent),
		mockMsgBrotker.EXPECT().SendToQueue(ctx, aws.SendToQueueInput{
			QueueName: "retry_queue", MessageBody: validBody,
		}).Return(nil),
	)

	mockRepo := repomocks.NewMockEvents(ctrl)
	gomock.InOrder(
		mockRepo.EXPECT().Insert(insertInput).Return(nil),
		mockRepo.EXPECT().Insert(insertInput).Return(aws.ErrInvalidInsertInput),
		mockRepo.EXPECT().Insert(insertInput).Return(aws.ErrInvalidInsertInput),
		mockRepo.EXPECT().Insert(insertInput).Return(nil),
		mockRepo.EXPECT().Insert(insertInput).Return(nil),
	)

	w := NewDispatchedEventProcessor(&EventProcessorInput{
		MsgBroker: mockMsgBrotker, Repo: mockRepo, QueueRetryProcess: "retry_queue",
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
			description: "should send reject event when body is invalid",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      bodyIncomplete,
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
			description: "should return err when insert fail and retry fail",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      validBody,
			},
			err: retryErr,
		},
		{
			description: "should send reject event when body is not json",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      "ok",
			},
			err: nil,
		},
		{
			description: "should return err when body is valid and fail publish event",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      validBody,
			},
			err: errSendEvent,
		},
		{
			description: "should return err when body is invalid and fail publish event",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      bodyIncomplete,
			},
			err: errSendEvent,
		},
		{
			description: "should send retry message when body is valid and fail publish event",
			eventRecord: models.EventRecord{
				MessageId: msgId,
				EventARN:  eventARN,
				Body:      validBody,
			},
			err: nil,
		},
	}
	for i, tc := range testCase {
		t.Run(fmt.Sprintf("TestProcessEvents_%d_%s", i, tc.description), func(t *testing.T) {
			err := w.ProcessEvents(context.Background(), &models.EventInput{
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
