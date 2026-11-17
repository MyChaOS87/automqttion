package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"

	"github.com/MyChaOS87/automqttion/pkg/hidden"
	loggerConfig "github.com/MyChaOS87/automqttion/pkg/log/config"
)

const (
	envPrefix    = "AUTOMQTTION"
	keyDelimiter = "::"
)

var (
	ErrConfigFileNotFound = errors.New("config file not found")
	ErrNilViper           = errors.New("config viper instance is nil")
)

// MqttTLSConfig configures TLS verification and optional client certificates for MQTT connections.
type MqttTLSConfig struct {
	Enabled            bool
	CAFile             string
	CertFile           string
	KeyFile            string
	ServerName         string
	InsecureSkipVerify bool
}

type MqttConfig struct {
	Broker   hidden.URL
	ClientID string
	Username string
	Password hidden.String
	TLS      MqttTLSConfig
}

type ActionConfig struct {
	Topic   string `yaml:"topic"`
	Content any    `yaml:"content"`
}

type MatchConfig struct {
	Match   any            `yaml:"match"`
	Actions []ActionConfig `yaml:"actions"`
}

// Config struct.
type Config struct {
	Logger   loggerConfig.Logger
	Mqtt     MqttConfig
	Automate map[string][]MatchConfig `mapstructure:"-"`
}

type automateSection struct {
	Automate map[string][]MatchConfig `yaml:"automate"`
}

// LoadConfig loads config file from given path.
func LoadConfig(filename string) (*viper.Viper, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter(keyDelimiter),
		viper.ExperimentalBindStruct(),
	)

	if filepath.Ext(filename) != "" {
		v.SetConfigFile(filename)
	} else {
		v.SetConfigName(filepath.Base(filename))
		v.AddConfigPath(filepath.Dir(filename))
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(keyDelimiter, "_"))
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok || errors.Is(err, os.ErrNotExist) {
			return nil, ErrConfigFileNotFound
		}

		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return v, nil
}

// ParseConfig parses config file.
func ParseConfig(v *viper.Viper) (*Config, error) {
	if v == nil {
		return nil, ErrNilViper
	}

	var c Config

	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfgFile := v.ConfigFileUsed(); cfgFile != "" {
		raw, err := os.ReadFile(filepath.Clean(cfgFile))
		if err != nil {
			return nil, fmt.Errorf("failed to read automate config %q: %w", cfgFile, err)
		}

		var section automateSection
		if err := yaml.Unmarshal(raw, &section); err != nil {
			return nil, fmt.Errorf("failed to parse automate config %q: %w", cfgFile, err)
		}

		c.Automate = section.Automate
	}

	return &c, nil
}
