package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	loggerConfig "github.com/MyChaOS87/automqttion/pkg/log/config"
)

const (
	envPrefix = "AUTOMQTTION"
)

var ErrConfigFileNotFound = errors.New("config file not found")

type MqttConfig struct {
	Broker string
}

type ActionConfig struct {
	Topic   string
	Content any
}

type MatchConfig struct {
	Match   any
	Actions []ActionConfig
}

// Config struct.
type Config struct {
	Logger   loggerConfig.Logger
	Mqtt     MqttConfig
	Automate map[string][]MatchConfig
}

// LoadConfig loads config file from given path.
func LoadConfig(filename string) (*viper.Viper, error) {
	v := viper.New()

	v.SetConfigName(filepath.Base(filename))
	v.AddConfigPath(filepath.Dir(filename))
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			return nil, ErrConfigFileNotFound
		}

		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return v, nil
}

// ParseConfig parses config file.
func ParseConfig(v *viper.Viper) (*Config, error) {
	var c Config

	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &c, nil
}
