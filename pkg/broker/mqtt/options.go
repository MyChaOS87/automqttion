package mqtt

import (
	"crypto/tls"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type (
	Option func(*Config)
	Config struct {
		clientOptions *mqtt.ClientOptions
	}
)

func Broker(broker string) Option {
	return func(c *Config) {
		c.clientOptions.AddBroker(broker)
	}
}

func TLS(tls *tls.Config) Option {
	return func(c *Config) {
		c.clientOptions.TLSConfig = tls
	}
}

func newDefaultConfig() *Config {
	return &Config{
		clientOptions: mqtt.NewClientOptions(),
	}
}
