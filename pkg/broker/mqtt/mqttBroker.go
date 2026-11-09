package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/MyChaOS87/automqttion/pkg/log"
)

type CallbackFunction[T any] func(topic string, data T)

type Topic struct {
	topic  string
	broker *Broker
}

type Broker struct {
	mu            sync.Mutex
	client        mqtt.Client
	connected     bool
	errorHandler  ErrorHandler
	subscriptions []func()
}

func (b *Broker) Topic(topicName string) *Topic {
	return &Topic{
		topic:  topicName,
		broker: b,
	}
}

func (b *Broker) Run(ctx context.Context) error {
	token := b.client.Connect()

	select {
	case <-token.Done():
		if err := token.Error(); err != nil {
			return fmt.Errorf("connect to mqtt broker: %w", err)
		}

		log.Infof("Connection to MQTT Broker established")
	case <-ctx.Done():
		b.disconnect()

		return fmt.Errorf("mqtt broker context cancelled during connect: %w", ctx.Err())
	}

	<-ctx.Done()
	b.disconnect()

	return nil
}

func (b *Broker) disconnect() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.connected = false
	b.client.Disconnect(0)
}

func (b *Broker) onConnect(_ mqtt.Client) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.connected = true

	log.Infof("MQTT client connected; active subscriptions: %d", len(b.subscriptions))

	for _, sub := range b.subscriptions {
		sub()
	}
}

func (b *Broker) onConnectionLost(_ mqtt.Client, err error) {
	b.mu.Lock()
	b.connected = false
	b.mu.Unlock()

	log.Warnf("Connection to MQTT Broker lost: %s", err)
}

func (b *Broker) publishInternal(topic, data string) {
	token := b.client.Publish(topic, 1, false, data)

	go func() {
		token.Wait()

		if err := token.Error(); err != nil {
			b.errorHandler(&Error{
				Op:      OpPublish,
				Topic:   topic,
				Payload: []byte(data),
				Err:     err,
			})
		} else {
			log.Infof("Successfully Published %s to %s", data, topic)
		}
	}()
}

func (b *Broker) registerSubscription(topic string, callback mqtt.MessageHandler) {
	sub := func() {
		token := b.client.Subscribe(topic, 0, callback)

		go func() {
			token.Wait()

			if err := token.Error(); err != nil {
				b.errorHandler(&Error{
					Op:    OpSubscribe,
					Topic: topic,
					Err:   err,
				})
			} else {
				log.Infof("Successfully Subscribed to %s", topic)
			}
		}()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscriptions = append(b.subscriptions, sub)

	if b.connected {
		sub()
	}
}

func (t *Topic) PublishString(s string) {
	t.broker.publishInternal(t.topic, s)
}

func (t *Topic) Publish(data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal publish payload for topic %q: %w", t.topic, err)
	}

	t.broker.publishInternal(t.topic, string(b))

	return nil
}

func (t *Topic) Subscribe[T any](callback CallbackFunction[T]) {
	t.broker.registerSubscription(t.topic, func(_ mqtt.Client, message mqtt.Message) {
		log.Debugf("got MQTT message: %s", message.Payload())

		var payload T

		err := json.Unmarshal(message.Payload(), &payload)
		if err != nil {
			t.broker.errorHandler(&Error{
				Op:      OpUnmarshal,
				Topic:   message.Topic(),
				Payload: message.Payload(),
				Err:     err,
			})

			return
		}

		log.Debugf("calling callback with: %#v", payload)
		callback(message.Topic(), payload)
	})
}

func NewBroker(options ...Option) *Broker {
	config := newDefaultConfig()

	for _, opt := range options {
		opt(config)
	}

	b := &Broker{
		errorHandler: config.errorHandler,
	}
	config.clientOptions.SetOnConnectHandler(b.onConnect)
	config.clientOptions.SetConnectionLostHandler(b.onConnectionLost)

	b.client = config.clientFactory(config.clientOptions)

	return b
}
