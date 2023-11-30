package null

import (
	"context"

	"github.com/MyChaOS87/automqttion.git/pkg/broker"
)

type (
	nullBroker struct{}
	nullTopic  struct{}
)

var (
	_ broker.Broker = &nullBroker{}
	_ broker.Topic  = &nullTopic{}
)

func (nullBroker) Run(context.Context, context.CancelFunc) {
}

func (nullBroker) Topic(string) broker.Topic {
	return &nullTopic{}
}

func (nullTopic) Publish(interface{}) {
}

func (nullTopic) PublishString(string) {}

func (nullTopic) Subscribe(hint interface{}, callback broker.CallbackFunction) {
}

func (nullTopic) SubscribeRawJson(callback broker.CallbackFunction) {
}

func NewBroker() broker.Broker {
	return &nullBroker{}
}
