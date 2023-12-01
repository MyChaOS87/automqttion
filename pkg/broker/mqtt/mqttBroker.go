package mqtt

import (
	"context"
	"encoding/json"
	"reflect"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/MyChaOS87/automqttion.git/pkg/broker"
	"github.com/MyChaOS87/automqttion.git/pkg/log"
)

var (
	_ broker.Broker = &mqttBroker{}
	_ broker.Topic  = &mqttTopic{}
)

type mqttTopic struct {
	topic  string
	client mqtt.Client
}

type mqttBroker struct {
	client mqtt.Client
}

func (p *mqttBroker) Topic(topicName string) broker.Topic {
	return &mqttTopic{
		topic:  topicName,
		client: p.client,
	}
}

func (p *mqttBroker) Run(ctx context.Context, cancel context.CancelFunc) {
	token := p.client.Connect()
	select {
	case <-token.Done():
		if err := token.Error(); err != nil {
			log.Errorf("Error Connecting to MQTT Broker: %s", err.Error())
			cancel()
		} else {
			log.Infof("Connection to MQTT Broker established")
		}
	case <-ctx.Done():
		p.client.Disconnect(0)
	}
}

func (t *mqttTopic) publishInternal(data string) {
	token := t.client.Publish(t.topic, 1, false, data)
	go func() {
		token.Wait()

		if err := token.Error(); err != nil {
			log.Errorf("Error on Publish to %s: %s", t.topic, err)
		} else {
			log.Infof("Successfully Published %s to %s", data, t.topic)
		}
	}()
}

func (t *mqttTopic) PublishString(s string) {
	t.publishInternal(s)
}

func (t *mqttTopic) Publish(data any) {
	b, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Error on Publish, cannot marshal JSON: %s", err)

		return
	}

	t.publishInternal(string(b))
}

func (t *mqttTopic) SubscribeRawJSON(callback broker.CallbackFunction) {
	internalCallback := func(client mqtt.Client, message mqtt.Message) {
		var raw any

		err := json.Unmarshal(message.Payload(), &raw)
		if err != nil {
			log.Errorf("Cannot unmarshal {%s}: %s", message.Payload(), err)

			return
		}

		log.Debugf("calling callback with: %#v", raw)
		callback(message.Topic(), raw)
	}

	t.subscribeInternal(internalCallback)
}

func (t *mqttTopic) Subscribe(hint any, callback broker.CallbackFunction) {
	internalCallback := func(client mqtt.Client, message mqtt.Message) {
		log.Debugf("got MQTT message: %s", message.Payload())

		payload := reflect.New(reflect.TypeOf(hint))

		err := json.Unmarshal(message.Payload(), payload.Interface())
		if err != nil {
			log.Errorf("Cannot unmarshal {%s} into %s: %s", message.Payload(), reflect.TypeOf(hint), err)

			return
		}

		log.Debugf("calling callback with: %#v", payload.Interface())
		callback(message.Topic(), payload.Interface())
	}

	t.subscribeInternal(internalCallback)
}

func (t *mqttTopic) subscribeInternal(internalCallback func(client mqtt.Client, message mqtt.Message)) {
	token := t.client.Subscribe(t.topic, 0, internalCallback)
	go func() {
		token.Wait()

		if err := token.Error(); err != nil {
			log.Errorf("Error on Subscribe to %s: %s", t.topic, err)
		} else {
			log.Infof("Successfully Subscribed to %s", t.topic)
		}
	}()
}

func NewBroker(options ...Option) broker.Broker {
	config := newDefaultConfig()

	for _, opt := range options {
		opt(config)
	}

	client := mqtt.NewClient(config.clientOptions)

	return &mqttBroker{
		client: client,
	}
}
