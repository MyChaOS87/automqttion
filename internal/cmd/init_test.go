package cmd_test

import (
	"context"
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/config"
	"github.com/MyChaOS87/automqttion/internal/cmd"
	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt"
	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt/mqtttest"
	"github.com/MyChaOS87/automqttion/pkg/hidden"
)

func TestInitWithConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupFile  func(t *testing.T, dir string) string
		wantErr    error
		wantErrAny bool
		wantBroker string
	}{
		{
			name: "missing config returns ErrConfigFileNotFound without panicking",
			setupFile: func(_ *testing.T, dir string) string {
				return filepath.Join(dir, "does-not-exist.yml")
			},
			wantErr: config.ErrConfigFileNotFound,
		},
		{
			name: "malformed YAML returns read error without panicking",
			setupFile: func(t *testing.T, dir string) string {
				t.Helper()

				p := filepath.Join(dir, "broken.yml")
				require.NoError(t, os.WriteFile(p, []byte("mqtt:\n  broker: [\n"), 0o600))

				return p
			},
			wantErrAny: true,
		},
		{
			name: "valid config initializes logger and signal context cleanly",
			setupFile: func(t *testing.T, dir string) string {
				t.Helper()

				p := filepath.Join(dir, "valid.yml")
				content := `---
mqtt:
  broker: tcp://user:s3cr3t@mosquitto.internal:1883
  password: another-secret
logger:
  development: true
  encoding: console
  level: info
`
				require.NoError(t, os.WriteFile(p, []byte(content), 0o600))

				return p
			},
			wantBroker: "tcp://user:s3cr3t@mosquitto.internal:1883",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := tt.setupFile(t, t.TempDir())
			parentCtx, parentCancel := context.WithCancel(t.Context())

			defer parentCancel()

			ctx, cancel, cfg, err := cmd.InitWithConfig(parentCtx, path)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, ctx)
				assert.Nil(t, cancel)
				assert.Nil(t, cfg)

				return
			}

			if tt.wantErrAny {
				require.Error(t, err)
				assert.Nil(t, ctx)
				assert.Nil(t, cancel)
				assert.Nil(t, cfg)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, ctx)
			require.NotNil(t, cancel)
			require.NotNil(t, cfg)

			defer cancel()

			assert.Equal(t, tt.wantBroker, cfg.Mqtt.Broker.Reveal())

			parentCancel()
			<-ctx.Done()
		})
	}
}

func TestNewMQTTBroker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      config.MqttConfig
		wantErr  error
		wantTLS  bool
		wantUser string
		wantPass string
		wantID   string
	}{
		{
			name: "wires ClientID, Credentials, and TLS automatically for tls:// scheme",
			cfg: config.MqttConfig{
				Broker:   hidden.URL("tls://mosquitto.internal:8883"),
				ClientID: "automqttion-unit",
				Username: "bridge",
				Password: hidden.String("s3cr3t"),
				TLS: config.MqttTLSConfig{
					ServerName: "mosquitto.internal",
				},
			},
			wantTLS:  true,
			wantUser: "bridge",
			wantPass: "s3cr3t",
			wantID:   "automqttion-unit",
		},
		{
			name: "plain tcp:// leaves TLSConfig nil when TLS disabled",
			cfg: config.MqttConfig{
				Broker: hidden.URL("tcp://mosquitto.internal:1883"),
			},
			wantTLS: false,
		},
		{
			name: "invalid TLS cert configuration returns ErrInvalidTLSConfig",
			cfg: config.MqttConfig{
				Broker: hidden.URL("tls://mosquitto.internal:8883"),
				TLS: config.MqttTLSConfig{
					Enabled:  true,
					CertFile: "/nonexistent/client.crt",
				},
			},
			wantErr: mqtt.ErrInvalidTLSConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sim := mqtttest.NewSimulator()
			brk, err := cmd.NewMQTTBroker(tt.cfg, sim.ClientFactory())

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, brk)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, brk)

			opts := sim.ClientOptions()
			require.NotNil(t, opts)
			assert.Equal(t, tt.wantID, opts.ClientID)
			assert.Equal(t, tt.wantUser, opts.Username)
			assert.Equal(t, tt.wantPass, opts.Password)

			if tt.wantTLS {
				require.NotNil(t, opts.TLSConfig)
				assert.Equal(t, uint16(tls.VersionTLS12), opts.TLSConfig.MinVersion)
				assert.Equal(t, tt.cfg.TLS.ServerName, opts.TLSConfig.ServerName)
			} else {
				assert.Nil(t, opts.TLSConfig)
			}
		})
	}
}
