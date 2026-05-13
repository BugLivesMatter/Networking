package messaging

// Имена сущностей RabbitMQ (ЛР8, точечная нотация для очередей).
const (
	ExchangeEvents           = "app.events"
	ExchangeDLX              = "app.dlx"
	RoutingKeyUserRegistered = "user.registered"
	QueueUserRegisteredDLQ   = "wp.auth.user.registered.dlq"
	SourceServiceName        = "wp-labs-api"
)
