package rabbitmq

import (
	"github.com/RichardKnop/machinery/v2"
	"github.com/RichardKnop/machinery/v2/tasks"
)

type Broker interface {
	RegisterTask(taskName string, handler interface{}) error
	EnqueueTask(taskName string, args ...interface{}) error
	GetServer() *machinery.Server
}

type RabbitMQBroker struct {
	server *machinery.Server
}

func (r *RabbitMQBroker) RegisterTask(taskName string, handler interface{}) error {
	return r.server.RegisterTask(taskName, handler)
}

func (r *RabbitMQBroker) EnqueueTask(taskName string, args ...interface{}) error {
	taskArgs := make([]tasks.Arg, len(args))
	for i, arg := range args {
		taskArgs[i] = tasks.Arg{
			Type:  "string",
			Value: arg,
		}
	}
	sig := &tasks.Signature{
		Name: taskName,
		Args: taskArgs,
	}
	_, err := r.server.SendTask(sig)
	return err
}

func (r *RabbitMQBroker) GetServer() *machinery.Server {
	return r.server
}
