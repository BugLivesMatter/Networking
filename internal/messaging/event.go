package messaging

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UserRegisteredPayload — данные события без секретов.
type UserRegisteredPayload struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

// EventMetadata — служебные поля сообщения.
type EventMetadata struct {
	Attempt       int    `json:"attempt"`
	SourceService string `json:"sourceService"`
}

// UserRegisteredMessage — контракт JSON для события user.registered.
type UserRegisteredMessage struct {
	EventID   string                `json:"eventId"`
	EventType string                `json:"eventType"`
	Timestamp string                `json:"timestamp"`
	Payload   UserRegisteredPayload `json:"payload"`
	Metadata  EventMetadata         `json:"metadata"`
}

// NewUserRegisteredMessage формирует первое сообщение (попытка 1).
func NewUserRegisteredMessage(userID uuid.UUID, email, displayName string) UserRegisteredMessage {
	if displayName == "" {
		displayName = email
	}
	return UserRegisteredMessage{
		EventID:   uuid.New().String(),
		EventType: "user.registered",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload: UserRegisteredPayload{
			UserID:      userID.String(),
			Email:       email,
			DisplayName: displayName,
		},
		Metadata: EventMetadata{
			Attempt:       1,
			SourceService: SourceServiceName,
		},
	}
}

// ToJSON сериализует сообщение в JSON для брокера.
func (m UserRegisteredMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
