package mqtttest

import (
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type token struct {
	done chan struct{}
	err  error
}

func newCompletedToken(err error) *token {
	ch := make(chan struct{})
	close(ch)

	return &token{
		done: ch,
		err:  err,
	}
}

func (t *token) Wait() bool {
	<-t.done

	return true
}

func (t *token) WaitTimeout(_ time.Duration) bool {
	<-t.done

	return true
}

func (t *token) Done() <-chan struct{} {
	return t.done
}

func (t *token) Error() error {
	return t.err
}

type message struct {
	topic   string
	payload []byte
}

func (m *message) Duplicate() bool   { return false }
func (m *message) Qos() byte         { return 0 }
func (m *message) Retained() bool    { return false }
func (m *message) Topic() string     { return m.topic }
func (m *message) MessageID() uint16 { return 1 }
func (m *message) Payload() []byte   { return m.payload }
func (m *message) Ack()              {}

type pahoClient struct {
	sim *Simulator
}

func (c *pahoClient) IsConnected() bool {
	c.sim.mu.Lock()
	defer c.sim.mu.Unlock()

	return c.sim.connected && !c.sim.disconnected
}

func (c *pahoClient) IsConnectionOpen() bool {
	return c.IsConnected()
}

func (c *pahoClient) Connect() paho.Token {
	c.sim.mu.Lock()
	block := c.sim.BlockConnect
	connectErr := c.sim.ConnectErr
	c.sim.mu.Unlock()

	if block {
		return &token{done: make(chan struct{})}
	}

	if connectErr != nil {
		return newCompletedToken(connectErr)
	}

	c.sim.mu.Lock()
	c.sim.connected = true
	deferOnConnect := c.sim.DeferOnConnect
	c.sim.mu.Unlock()

	if !deferOnConnect {
		c.sim.SimulateConnect()
	}

	return newCompletedToken(nil)
}

func (c *pahoClient) Disconnect(_ uint) {
	c.sim.mu.Lock()
	defer c.sim.mu.Unlock()

	c.sim.connected = false
	c.sim.disconnected = true
	c.sim.subs = nil
}

func (c *pahoClient) Publish(topic string, qos byte, retained bool, payload any) paho.Token {
	var s string

	switch v := payload.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	}

	c.sim.mu.Lock()
	c.sim.pubs = append(c.sim.pubs, PublishedMessage{
		Topic:   topic,
		QoS:     qos,
		Retain:  retained,
		Payload: s,
	})
	err := c.sim.PubErr
	c.sim.mu.Unlock()

	return newCompletedToken(err)
}

func (c *pahoClient) Subscribe(topic string, qos byte, callback paho.MessageHandler) paho.Token {
	c.sim.mu.Lock()
	c.sim.subCalls = append(c.sim.subCalls, topic)

	err := c.sim.SubErr
	if err == nil {
		c.sim.subs = append(c.sim.subs, subRecord{
			topic:    topic,
			qos:      qos,
			callback: callback,
		})
	}
	c.sim.mu.Unlock()

	return newCompletedToken(err)
}

func (c *pahoClient) SubscribeMultiple(_ map[string]byte, _ paho.MessageHandler) paho.Token {
	return newCompletedToken(nil)
}

func (c *pahoClient) Unsubscribe(_ ...string) paho.Token {
	return newCompletedToken(nil)
}

func (c *pahoClient) AddRoute(_ string, _ paho.MessageHandler) {}

func (c *pahoClient) OptionsReader() paho.ClientOptionsReader {
	return paho.ClientOptionsReader{}
}
