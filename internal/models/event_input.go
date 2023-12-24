package models

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type (
	EventInput struct {
		Records []EventRecord `json:"Records"`
	}

	EventRecord struct {
		MessageId string `json:"messageId"`
		Body      string `json:"body"`
		EventARN  string `json:"eventSourceARN"`
	}

	EventMessageBody struct {
		EventId string `json:"event_id"`
		Context string `json:"context"`
		Type    string `json:"type"`
		Tenant  string `json:"tenant"`
		Data    string `json:"data"`
	}

	MessageBody interface {
		Validate() error
	}
)

func (s *EventMessageBody) Validate() error {
	return validation.ValidateStruct(s,
		validation.Field(&s.EventId, validation.Required, is.UUID),
		validation.Field(&s.Context, validation.Required),
		validation.Field(&s.Type, validation.Required),
		validation.Field(&s.Tenant, validation.Required, is.UUID),
		validation.Field(&s.Data, validation.Required, is.JSON),
	)
}
