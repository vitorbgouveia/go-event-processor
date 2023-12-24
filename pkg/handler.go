package pkg

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/vitorbgouveia/go-event-processor/internal/models"
	"github.com/vitorbgouveia/go-event-processor/internal/repository"
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
	"github.com/vitorbgouveia/go-event-processor/pkg/worker"
)

type (
	lambdaHandler struct {
		queueRetryProcess   string
		queueDLQProcess     string
		MsgBroker           aws.MessageBroker
		DispatchedEventRepo repository.Events
	}

	LambdaHandler interface {
		Handle(ctx context.Context, event *models.EventInput) error
	}

	LambdaHandlerInput struct {
		QueueRetryProcess   string
		QueueDLQProcess     string
		MsgBroker           aws.MessageBroker
		DispatchedEventRepo repository.Events
	}
)

func NewLambdaHandler(i LambdaHandlerInput) LambdaHandler {
	return &lambdaHandler{
		i.QueueRetryProcess, i.QueueDLQProcess, i.MsgBroker, i.DispatchedEventRepo,
	}
}

func (s lambdaHandler) Handle(ctx context.Context, event *models.EventInput) error {
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w := worker.NewDispatchedEventProcessor(&worker.EventProcessorInput{
		MsgBroker: s.MsgBroker, QueueRetryProcess: s.queueRetryProcess, Repo: s.DispatchedEventRepo,
	})
	if err := w.ProcessEvents(ctx, event, sigCtx.Done()); err != nil {
		return err
	}

	return nil
}
