package broker

import "context"

type Broker interface {
	Run(ctx context.Context, cancel context.CancelFunc)
	Topic(topic string) Topic
}

type Topic interface {
	PublishString(s string)
	Publish(i any)
	Subscribe(hint any, callback CallbackFunction)
	SubscribeRawJSON(callback CallbackFunction)
}

type CallbackFunction func(topic string, data any)
