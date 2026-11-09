package mqtttest

import (
	"slices"
	"sync"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt"
)

type PublishedMessage struct {
	Topic   string
	QoS     byte
	Retain  bool
	Payload string
}

type subRecord struct {
	topic    string
	qos      byte
	callback paho.MessageHandler
}

type Simulator struct {
	mu             sync.Mutex
	client         *pahoClient
	opts           *paho.ClientOptions
	ConnectErr     error
	BlockConnect   bool
	DeferOnConnect bool
	SubErr         error
	PubErr         error
	connected      bool
	disconnected   bool
	subCalls       []string
	subs           []subRecord
	pubs           []PublishedMessage
}

func NewSimulator() *Simulator {
	sim := &Simulator{}
	sim.client = &pahoClient{sim: sim}

	return sim
}

func NewBroker(options ...mqtt.Option) (*mqtt.Broker, *Simulator) {
	sim := NewSimulator()
	opts := make([]mqtt.Option, 0, len(options)+1)
	opts = append(opts, sim.ClientFactory())
	opts = append(opts, options...)

	return mqtt.NewBroker(opts...), sim
}

func (s *Simulator) ClientFactory() mqtt.Option {
	return mqtt.ClientFactory(func(opts *paho.ClientOptions) paho.Client {
		s.mu.Lock()
		s.opts = opts
		s.mu.Unlock()

		return s.client
	})
}

func (s *Simulator) connect() {
	s.mu.Lock()
	if s.disconnected {
		s.mu.Unlock()

		return
	}

	s.connected = true
	opts := s.opts
	client := s.client
	s.mu.Unlock()

	if opts != nil && opts.OnConnect != nil {
		opts.OnConnect(client)
	}
}

func (s *Simulator) disconnect(err error) {
	s.mu.Lock()
	s.connected = false
	s.subs = nil
	opts := s.opts
	client := s.client
	s.mu.Unlock()

	if opts != nil && opts.OnConnectionLost != nil {
		opts.OnConnectionLost(client, err)
	}
}

func (s *Simulator) SimulateConnect() {
	go s.connect()
}

func (s *Simulator) SimulateDisconnect(err error) {
	go s.disconnect(err)
}

func (s *Simulator) SimulateReconnect(err error) {
	go func() {
		s.disconnect(err)
		s.connect()
	}()
}

func (s *Simulator) Deliver(topic, payload string) bool {
	s.mu.Lock()

	var handlers []paho.MessageHandler

	for _, sub := range s.subs {
		if sub.topic == topic && sub.callback != nil {
			handlers = append(handlers, sub.callback)
		}
	}

	client := s.client
	s.mu.Unlock()

	if len(handlers) == 0 {
		return false
	}

	msg := &message{
		topic:   topic,
		payload: []byte(payload),
	}

	for _, h := range handlers {
		h(client, msg)
	}

	return true
}

func (s *Simulator) SubscribedTopics() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.subCalls)
}

func (s *Simulator) PublishedMessages() []PublishedMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.pubs)
}

func (s *Simulator) Disconnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.disconnected
}
