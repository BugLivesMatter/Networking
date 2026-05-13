package messaging

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

// Publisher публикует JSON в exchange app.events (канал не потокобезопасен — mutex).
type Publisher struct {
	mu sync.Mutex
	ch *amqp091.Channel
}

func NewPublisher(ch *amqp091.Channel) *Publisher {
	return &Publisher{ch: ch}
}

// RegistrationEventPublisher публикует событие регистрации (инъекция в auth).
type RegistrationEventPublisher interface {
	PublishUserRegistered(ctx context.Context, userID uuid.UUID, email, displayName string) (eventID string, err error)
}

// PublishUserRegistered публикует событие по данным пользователя (возвращает eventId для логов).
func (p *Publisher) PublishUserRegistered(ctx context.Context, userID uuid.UUID, email, displayName string) (string, error) {
	msg := NewUserRegisteredMessage(userID, email, displayName)
	if err := p.PublishUserRegisteredMessage(ctx, msg); err != nil {
		return "", err
	}
	return msg.EventID, nil
}

// PublishUserRegisteredMessage публикует готовое сообщение.
func (p *Publisher) PublishUserRegisteredMessage(ctx context.Context, msg UserRegisteredMessage) error {
	body, err := msg.ToJSON()
	if err != nil {
		return err
	}
	return p.PublishJSON(ctx, body)
}

// PublishJSON публикует произвольный JSON (для повторной публикации с тем же eventId).
func (p *Publisher) PublishJSON(ctx context.Context, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ch.PublishWithContext(ctx,
		ExchangeEvents,
		RoutingKeyUserRegistered,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			Body:         body,
		},
	)
}

// DialAMQP открывает соединение с RabbitMQ.
func DialAMQP(url string) (*amqp091.Connection, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("подключение к RabbitMQ: %w", err)
	}
	return conn, nil
}
