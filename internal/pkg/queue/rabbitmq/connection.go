package rabbitmq

import (
	"context"

	"github.com/RichardKnop/machinery/v2"
	"github.com/RichardKnop/machinery/v2/config"
	amqpBroker "github.com/RichardKnop/machinery/v2/brokers/amqp"
	nullbackend "github.com/RichardKnop/machinery/v2/backends/null"
	eagerlock "github.com/RichardKnop/machinery/v2/locks/eager"
)

func New(ctx context.Context, url string) Broker {
	cnf := &config.Config{
		Broker:          url,
		DefaultQueue:    "docstore_tasks",
		ResultBackend:   url,
		ResultsExpireIn: 3600,
		AMQP: &config.AMQPConfig{
			Exchange:      "docstore",
			ExchangeType:  "direct",
			BindingKey:    "machinery_task",
			PrefetchCount: 3,
		},
	}
	broker := amqpBroker.New(cnf)
	backend := nullbackend.New()
	lock := eagerlock.New()
	server := machinery.NewServer(cnf, broker, backend, lock)

	return &RabbitMQBroker{
		server: server,
	}
}