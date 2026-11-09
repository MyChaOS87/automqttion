package mqtt

import (
	"crypto/tls"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type (
	Option func(*Config)
	Config struct {
		clientOptions *mqtt.ClientOptions
		clientFactory func(*mqtt.ClientOptions) mqtt.Client
		errorHandler  ErrorHandler
	}
)

func BrokerURL(broker string) Option {
	return func(c *Config) {
		c.clientOptions.AddBroker(broker)
	}
}

func TLS(tls *tls.Config) Option {
	return func(c *Config) {
		c.clientOptions.TLSConfig = tls
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
	}
}
