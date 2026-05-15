package messaging

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	amqp091 "github.com/rabbitmq/amqp091-go"

	"github.com/lab2/rest-api/internal/cache"
	"github.com/lab2/rest-api/internal/email"
)

const eventProcessedTTL = 24 * time.Hour

// RunUserRegisteredConsumer обрабатывает очередь регистраций в фоне (ack после успешной отправки SMTP).
func RunUserRegisteredConsumer(
	ctx context.Context,
	ch *amqp091.Channel,
	mainQueue string,
	cacheSvc cache.Service,
	mailer *email.Sender,
	pub *Publisher,
) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("консьюмер RabbitMQ: паника при обработке: %v", r)
		}
	}()

	if err := ch.Qos(1, 0, false); err != nil {
		log.Printf("консьюмер RabbitMQ: Qos: %v", err)
		return
	}

	msgs, err := ch.Consume(mainQueue, "user-registered-worker", false, false, false, false, nil)
	if err != nil {
		log.Printf("консьюмер RabbitMQ: Consume: %v", err)
		return
	}

	log.Printf("консьюмер RabbitMQ: ожидание сообщений в очереди %q", mainQueue)

	for {
		select {
		case <-ctx.Done():
			log.Printf("консьюмер RabbitMQ: остановка по контексту")
			return
		case d, ok := <-msgs:
			if !ok {
				log.Printf("консьюмер RabbitMQ: канал доставки закрыт")
				return
			}
			handleUserRegisteredDelivery(ctx, d, cacheSvc, mailer, pub)
		}
	}
}

func handleUserRegisteredDelivery(
	ctx context.Context,
	d amqp091.Delivery,
	cacheSvc cache.Service,
	mailer *email.Sender,
	pub *Publisher,
) {
	var msg UserRegisteredMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		log.Printf("консьюмер RabbitMQ: некорректный JSON, отклонение сообщения: %v", err)
		_ = d.Nack(false, false)
		return
	}
	if msg.EventType != "user.registered" {
		log.Printf("консьюмер RabbitMQ: неизвестный eventType=%q", msg.EventType)
		_ = d.Nack(false, false)
		return
	}

	processedKey := cache.EventProcessedKey(msg.EventID)
	exists, err := cacheSvc.Exists(ctx, processedKey)
	if err != nil {
		log.Printf("консьюмер RabbitMQ: ошибка Redis при проверке идемпотентности eventId=%s: %v", msg.EventID, err)
		_ = d.Nack(true, true)
		return
	}
	if exists {
		log.Printf("консьюмер RabbitMQ: событие уже обработано, пропуск (eventId=%s)", msg.EventID)
		_ = d.Ack(false)
		return
	}

	userID, err := uuid.Parse(msg.Payload.UserID)
	if err != nil {
		log.Printf("консьюмер RabbitMQ: некорректный userId в сообщении eventId=%s", msg.EventID)
		_ = d.Nack(false, false)
		return
	}

	log.Printf("консьюмер RabbitMQ: попытка отправки письма eventId=%s attempt=%d userId=%s",
		msg.EventID, msg.Metadata.Attempt, userID)

	sendErr := mailer.SendWelcome(ctx, msg.Payload.Email, msg.Payload.DisplayName, userID)
	if sendErr == nil {
		if err := cacheSvc.Set(ctx, processedKey, map[string]string{"ok": "1"}, eventProcessedTTL); err != nil {
			log.Printf("консьюмер RabbitMQ: письмо отправлено, но не удалось записать идемпотентность в Redis: %v", err)
		}
		_ = d.Ack(false)
		log.Printf("консьюмер RabbitMQ: письмо успешно отправлено eventId=%s", msg.EventID)
		return
	}

	log.Printf("консьюмер RabbitMQ: ошибка SMTP eventId=%s attempt=%d: %v", msg.EventID, msg.Metadata.Attempt, sendErr)

	if msg.Metadata.Attempt >= 3 {
		_ = d.Nack(false, false)
		log.Printf("консьюмер RabbitMQ: исчерпаны попытки, сообщение отправлено в DLQ (eventId=%s)", msg.EventID)
		return
	}

	msg.Metadata.Attempt++
	body, mErr := msg.ToJSON()
	if mErr != nil {
		log.Printf("консьюмер RabbitMQ: сериализация повторной попытки: %v", mErr)
		_ = d.Nack(false, false)
		return
	}
	if err := pub.PublishJSON(ctx, body); err != nil {
		log.Printf("консьюмер RabbitMQ: не удалось опубликовать повтор eventId=%s: %v", msg.EventID, err)
		_ = d.Nack(true, true)
		return
	}
	_ = d.Ack(false)
	log.Printf("консьюмер RabbitMQ: запланирована повторная попытка eventId=%s nextAttempt=%d", msg.EventID, msg.Metadata.Attempt)
}
