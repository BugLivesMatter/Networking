package messaging

import (
	"fmt"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

// DeclareTopology объявляет обменники, очереди и привязки (все durable).
func DeclareTopology(ch *amqp091.Channel, mainQueueName string) error {
	if err := ch.ExchangeDeclare(ExchangeDLX, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("объявление exchange %s: %w", ExchangeDLX, err)
	}
	if err := ch.ExchangeDeclare(ExchangeEvents, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("объявление exchange %s: %w", ExchangeEvents, err)
	}

	argsMain := amqp091.Table{
		"x-dead-letter-exchange":    ExchangeDLX,
		"x-dead-letter-routing-key": RoutingKeyUserRegistered,
	}
	_, err := ch.QueueDeclare(mainQueueName, true, false, false, false, argsMain)
	if err != nil {
		return fmt.Errorf("объявление очереди %s: %w", mainQueueName, err)
	}
	if err := ch.QueueBind(mainQueueName, RoutingKeyUserRegistered, ExchangeEvents, false, nil); err != nil {
		return fmt.Errorf("привязка очереди %s к %s: %w", mainQueueName, ExchangeEvents, err)
	}

	_, err = ch.QueueDeclare(QueueUserRegisteredDLQ, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("объявление DLQ %s: %w", QueueUserRegisteredDLQ, err)
	}
	if err := ch.QueueBind(QueueUserRegisteredDLQ, RoutingKeyUserRegistered, ExchangeDLX, false, nil); err != nil {
		return fmt.Errorf("привязка DLQ к %s: %w", ExchangeDLX, err)
	}
	return nil
}
