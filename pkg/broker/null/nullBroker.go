package null

import (
	"context"

	"github.com/MyChaOS87/automqttion/pkg/broker"
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

func (nullTopic) Publish(any) {
}

func (nullTopic) PublishString(string) {}

func (nullTopic) Subscribe(_ any, _ broker.CallbackFunction) {
}

func (nullTopic) SubscribeRawJSON(_ broker.CallbackFunction) {
}

func NewBroker() broker.Broker {
	return &nullBroker{}
}
