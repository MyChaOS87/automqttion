package cmd

import (
	"fmt"
	"strings"

	"github.com/MyChaOS87/automqttion/config"
	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt"
)

// NewMQTTBroker constructs a *mqtt.Broker configured with credentials, client ID, and optional TLS.
func NewMQTTBroker(cfg config.MqttConfig, extraOpts ...mqtt.Option) (*mqtt.Broker, error) {
	brokerURL := cfg.Broker.Reveal()
	opts := []mqtt.Option{
		mqtt.BrokerURL(brokerURL),
		mqtt.ClientID(cfg.ClientID),
		mqtt.Credentials(cfg.Username, cfg.Password.Reveal()),
	}

	if tlsEnabled(brokerURL, cfg.TLS) {
		tlsCfg, err := mqtt.BuildTLSConfig(mqtt.TLSConfig{
			Enabled:            true,
			CAFile:             cfg.TLS.CAFile,
			CertFile:           cfg.TLS.CertFile,
			KeyFile:            cfg.TLS.KeyFile,
			ServerName:         cfg.TLS.ServerName,
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		})
		if err != nil {
			return nil, fmt.Errorf("build mqtt tls config: %w", err)
		}

		opts = append(opts, mqtt.TLS(tlsCfg))
	}

	opts = append(opts, extraOpts...)

	return mqtt.NewBroker(opts...), nil
}

func tlsEnabled(brokerURL string, cfg config.MqttTLSConfig) bool {
	if cfg.Enabled || cfg.CAFile != "" || cfg.CertFile != "" || cfg.KeyFile != "" {
		return true
	}

	lower := strings.ToLower(brokerURL)

	return strings.HasPrefix(lower, "ssl://") ||
		strings.HasPrefix(lower, "tls://") ||
		strings.HasPrefix(lower, "wss://")
}
