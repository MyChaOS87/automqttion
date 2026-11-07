package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/config"
)

type loadAndParseTestCases []struct {
	name           string
	configName     string
	yamlContent    string
	expectedBroker string
	expectedLevel  string
	expectedTopics int
	expectedErr    error
	expectReadErr  bool
	expectParseErr bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.yamlContent != "" {
				filename := tt.configName + ".yml"
				err := os.WriteFile(filename, []byte(tt.yamlContent), 0o600)
				require.NoError(t, err)
				t.Cleanup(func() {
					_ = os.Remove(filename)
				})
			}

			v, err := config.LoadConfig(tt.configName)
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

			assert.Equal(t, tt.expectedBroker, cfg.Mqtt.Broker)
			assert.Equal(t, tt.expectedLevel, cfg.Logger.Level)
			assert.Len(t, cfg.Automate, tt.expectedTopics)
		})
	}
}
