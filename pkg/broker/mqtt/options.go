package mqtt

import (
	"crypto/tls"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	defaultTokenTimeout        = 10 * time.Second
	defaultDisconnectQuiesceMs = 250
)

type (
	Option func(*Config)
	Config struct {
		clientOptions *mqtt.ClientOptions
		clientFactory func(*mqtt.ClientOptions) mqtt.Client
		errorHandler  ErrorHandler
		tokenTimeout  time.Duration
	}
)

func BrokerURL(broker string) Option {
	return func(c *Config) {
		c.clientOptions.AddBroker(broker)
	}
}

// ClientID configures the MQTT client identifier when non-empty.
func ClientID(clientID string) Option {
	return func(c *Config) {
		if clientID != "" {
			c.clientOptions.SetClientID(clientID)
		}
	}
}

// Credentials configures MQTT username and password authentication when non-empty.
func Credentials(username, password string) Option {
	return func(c *Config) {
		if username != "" {
			c.clientOptions.SetUsername(username)
		}

		if password != "" {
			c.clientOptions.SetPassword(password)
		}
	}
}

func TLS(tls *tls.Config) Option {
	return func(c *Config) {
		c.clientOptions.TLSConfig = tls
	}
}

// TokenTimeout sets the maximum duration to wait for async publish/subscribe tokens.
func TokenTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		if timeout > 0 {
			c.tokenTimeout = timeout
		}
	}
}

func ClientFactory(factory func(*mqtt.ClientOptions) mqtt.Client) Option {
	return func(c *Config) {
		if factory != nil {
			c.clientFactory = factory
		}
	}
}

func OnError(handler ErrorHandler) Option {
	return func(c *Config) {
		if handler != nil {
			c.errorHandler = handler
		}
	}
}

func newDefaultConfig() *Config {
	return &Config{
		clientOptions: mqtt.NewClientOptions(),
		clientFactory: mqtt.NewClient,
		errorHandler:  defaultErrorHandler,
		tokenTimeout:  defaultTokenTimeout,
	}
}
