package contracts

import (
	"fmt"
)

type DispatchedEvent struct {
	Records []records `json:"Records"`
}

type records struct {
	MessageId string `json:"messageId"`
	Body      string `json:"body"`
	EventARN  string `json:"eventSourceARN"`
}

type MessageBody interface {
	Validate() error
}

type EventMessageBody struct {
	Context   string `json:"context"`
	Type      string `json:"type"`
	Tenant    string `json:"tenant"`
	EventData string `json:"event_data"`
}

func (s *EventMessageBody) Validate() error {
	switch {
	case s.Context == "":
		return fmt.Errorf("event context is empty: %s", s.Context)
	case s.Type == "":
		return fmt.Errorf("event type is empty: %s", s.Type)
	case s.Tenant == "":
		return fmt.Errorf("event tenant is empty: %s", s.Tenant)
	case s.EventData == "":
		return fmt.Errorf("event data is empty: %s", s.EventData)
	}

	return nil
}
