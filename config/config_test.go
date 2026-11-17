package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/config"
	"github.com/MyChaOS87/automqttion/pkg/hidden"
)

type loadAndParseTestCases []struct {
	name             string
	configName       string
	fileExt          string
	yamlContent      string
	expectedBroker   string
	expectedLevel    string
	expectedTopics   int
	expectedAutomate map[string][]config.MatchConfig
	expectedErr      error
	expectReadErr    bool
	expectParseErr   bool
}

func TestLoadAndParseConfig(t *testing.T) {
	t.Parallel()

	tests := loadAndParseTestCases{
		{
			name:           "repository default config.yml loads and parses",
			configName:     "config",
			expectedBroker: "tcp://mosquitto.internal:1883",
			expectedLevel:  "info",
			expectedTopics: 1,
		},
		{
			name:        "non-existent configuration file returns ErrConfigFileNotFound",
			configName:  "nonexistent_config_file",
			expectedErr: config.ErrConfigFileNotFound,
		},
		{
			name:        "non-existent explicit yml path returns ErrConfigFileNotFound",
			configName:  filepath.Join(t.TempDir(), "missing.yml"),
			expectedErr: config.ErrConfigFileNotFound,
		},
		{
			name:          "invalid YAML syntax returns read error",
			configName:    "invalid_syntax_test",
			yamlContent:   "mqtt:\n  broker: [unclosed\n",
			expectReadErr: true,
		},
		{
			name:           "incompatible schema type returns parse error",
			configName:     "invalid_schema_test",
			yamlContent:    "automate: not-a-map\n",
			expectParseErr: true,
		},
		{
			name:    "mixed-case and dotted MQTT topics preserve exact keys",
			fileExt: ".yml",
			yamlContent: `---
mqtt:
  broker: tcp://broker.local:1883
automate:
  zigbee2mqtt/KitchenSwitch:
    - match:
        PowerState: ON
        sensor.temp: 21
      actions:
        - topic: zigbee2mqtt/HallwayLight/set
          content:
            $json:
              TargetState: ON
              meta.source: KitchenSwitch
  tele/Tasmota_01/SENSOR:
    - match:
        ENERGY.Power: 42
      actions:
        - topic: home/sensor.1/alert
          content:
            $plain: HIGH
  home/sensor.1/state:
    - match: on
      actions:
        - topic: v1.0/events
          content:
            $plain: ack
  home/sensor:
    - match: off
      actions:
        - topic: v1.0/events
          content:
            $plain: nack
logger:
  level: warn
`,
			expectedBroker: "tcp://broker.local:1883",
			expectedLevel:  "warn",
			expectedTopics: 4,
			expectedAutomate: map[string][]config.MatchConfig{
				"zigbee2mqtt/KitchenSwitch": {
					{
						Match: map[string]any{"PowerState": "ON", "sensor.temp": 21},
						Actions: []config.ActionConfig{
							{
								Topic: "zigbee2mqtt/HallwayLight/set",
								Content: map[string]any{
									"$json": map[string]any{"TargetState": "ON", "meta.source": "KitchenSwitch"},
								},
							},
						},
					},
				},
				"tele/Tasmota_01/SENSOR": {
					{
						Match: map[string]any{"ENERGY.Power": 42},
						Actions: []config.ActionConfig{
							{
								Topic:   "home/sensor.1/alert",
								Content: map[string]any{"$plain": "HIGH"},
							},
						},
					},
				},
				"home/sensor.1/state": {
					{
						Match: "on",
						Actions: []config.ActionConfig{
							{
								Topic:   "v1.0/events",
								Content: map[string]any{"$plain": "ack"},
							},
						},
					},
				},
				"home/sensor": {
					{
						Match: "off",
						Actions: []config.ActionConfig{
							{
								Topic:   "v1.0/events",
								Content: map[string]any{"$plain": "nack"},
							},
						},
					},
				},
			},
		},
		{
			name:       "explicit .yaml file path loads and preserves dotted topic",
			configName: "explicit_config",
			fileExt:    ".yaml",
			yamlContent: `mqtt:
  broker: ssl://broker.example:8883
automate:
  v1.0/events:
    - match: online
      actions:
        - topic: status/Out.1
          content:
            $plain: ready
logger:
  level: debug
`,
			expectedBroker: "ssl://broker.example:8883",
			expectedLevel:  "debug",
			expectedTopics: 1,
			expectedAutomate: map[string][]config.MatchConfig{
				"v1.0/events": {
					{
						Match: "online",
						Actions: []config.ActionConfig{
							{
								Topic:   "status/Out.1",
								Content: map[string]any{"$plain": "ready"},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configName := tt.configName
			if tt.yamlContent != "" {
				ext := tt.fileExt
				if ext == "" {
					ext = ".yml"
				}

				base := tt.configName
				if base == "" {
					base = "config"
				}

				filePath := filepath.Join(t.TempDir(), base+ext)
				err := os.WriteFile(filePath, []byte(tt.yamlContent), 0o600)
				require.NoError(t, err)

				if tt.fileExt != "" {
					configName = filePath
				} else {
					configName = filepath.Join(filepath.Dir(filePath), base)
				}
			}

			v, err := config.LoadConfig(configName)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				assert.Nil(t, v)

				return
			}

			if tt.expectReadErr {
				require.Error(t, err)
				require.NotErrorIs(t, err, config.ErrConfigFileNotFound)
				assert.Nil(t, v)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, v)

			cfg, err := config.ParseConfig(v)
			if tt.expectParseErr {
				require.Error(t, err)
				assert.Nil(t, cfg)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			assert.Equal(t, tt.expectedBroker, cfg.Mqtt.Broker.Reveal())
			assert.Equal(t, tt.expectedLevel, cfg.Logger.Level)
			assert.Len(t, cfg.Automate, tt.expectedTopics)

			if tt.expectedAutomate != nil {
				assert.Equal(t, tt.expectedAutomate, cfg.Automate)
			}
		})
	}
}

func TestParseConfigNilViper(t *testing.T) {
	t.Parallel()

	cfg, err := config.ParseConfig(nil)
	require.ErrorIs(t, err, config.ErrNilViper)
	assert.Nil(t, cfg)
}

type envOverrideTestCases []struct {
	name     string
	env      map[string]string
	assertFn func(t *testing.T, cfg *config.Config)
}

func TestParseConfigEnvOverrides(t *testing.T) {
	tests := envOverrideTestCases{
		{
			name: "MQTT and Logger env vars override YAML and omitted fields",
			env: map[string]string{
				"AUTOMQTTION_MQTT_BROKER":                 "ssl://svc:inline-secret@override.internal:8883",
				"AUTOMQTTION_MQTT_CLIENTID":               "automqttion-prod-1",
				"AUTOMQTTION_MQTT_USERNAME":               "mqtt-user",
				"AUTOMQTTION_MQTT_PASSWORD":               "mqtt-secret",
				"AUTOMQTTION_MQTT_TLS_ENABLED":            "true",
				"AUTOMQTTION_MQTT_TLS_CAFILE":             "/certs/ca.crt",
				"AUTOMQTTION_MQTT_TLS_CERTFILE":           "/certs/client.crt",
				"AUTOMQTTION_MQTT_TLS_KEYFILE":            "/certs/client.key",
				"AUTOMQTTION_MQTT_TLS_SERVERNAME":         "override.internal",
				"AUTOMQTTION_MQTT_TLS_INSECURESKIPVERIFY": "true",
				"AUTOMQTTION_LOGGER_DEVELOPMENT":          "false",
				"AUTOMQTTION_LOGGER_ENCODING":             "json",
				"AUTOMQTTION_LOGGER_LEVEL":                "error",
			},
			assertFn: func(t *testing.T, cfg *config.Config) {
				t.Helper()

				const wantMaskedBroker = "ssl://svc:" + hidden.HiddenText + "@override.internal:8883"

				assert.Equal(t, "ssl://svc:inline-secret@override.internal:8883", cfg.Mqtt.Broker.Reveal())
				assert.Equal(t, "automqttion-prod-1", cfg.Mqtt.ClientID)
				assert.Equal(t, "mqtt-user", cfg.Mqtt.Username)
				assert.Equal(t, "mqtt-secret", cfg.Mqtt.Password.Reveal())
				assert.True(t, cfg.Mqtt.TLS.Enabled)
				assert.Equal(t, "/certs/ca.crt", cfg.Mqtt.TLS.CAFile)
				assert.Equal(t, "/certs/client.crt", cfg.Mqtt.TLS.CertFile)
				assert.Equal(t, "/certs/client.key", cfg.Mqtt.TLS.KeyFile)
				assert.Equal(t, "override.internal", cfg.Mqtt.TLS.ServerName)
				assert.True(t, cfg.Mqtt.TLS.InsecureSkipVerify)
				assert.False(t, cfg.Logger.Development)
				assert.Equal(t, "json", cfg.Logger.Encoding)
				assert.Equal(t, "error", cfg.Logger.Level)
				assert.Contains(t, cfg.Automate, "zigbee2mqtt/KitchenSwitch")

				formattedPlusV := fmt.Sprintf("%+v", cfg)
				formattedSharpV := fmt.Sprintf("%#v", cfg)

				assert.NotContains(t, formattedPlusV, "mqtt-secret")
				assert.NotContains(t, formattedPlusV, "inline-secret")
				assert.Contains(t, formattedPlusV, hidden.HiddenText)
				assert.Contains(t, formattedPlusV, wantMaskedBroker)

				assert.NotContains(t, formattedSharpV, "mqtt-secret")
				assert.NotContains(t, formattedSharpV, "inline-secret")
				assert.Contains(t, formattedSharpV, hidden.HiddenText)
			},
		},
	}

	const baseYAML = `mqtt:
  broker: tcp://base.internal:1883
automate:
  zigbee2mqtt/KitchenSwitch:
    - match:
        State: ON
      actions:
        - topic: zigbee2mqtt/Lamp/set
          content:
            $plain: ON
logger:
  development: true
  encoding: console
  level: info
`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgPath := filepath.Join(t.TempDir(), "config.yml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(baseYAML), 0o600))

			for k, val := range tt.env {
				t.Setenv(k, val)
			}

			v, err := config.LoadConfig(cfgPath)
			require.NoError(t, err)

			cfg, err := config.ParseConfig(v)
			require.NoError(t, err)
			require.NotNil(t, cfg)

			tt.assertFn(t, cfg)
		})
	}
}
