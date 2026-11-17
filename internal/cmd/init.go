package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MyChaOS87/automqttion/config"
	logger "github.com/MyChaOS87/automqttion/pkg/log"
)

const (
	configFilename = "./config/config"
)

// Init loads the default configuration, initializes logging, and returns a signal-aware context.
func Init() (context.Context, context.CancelFunc, *config.Config, error) {
	return InitWithConfig(context.Background(), configFilename)
}

// InitWithConfig loads configuration from configPath, initializes logging, and listens for shutdown signals.
func InitWithConfig(
	parentCtx context.Context,
	configPath string,
) (context.Context, context.CancelFunc, *config.Config, error) {
	log.Default().SetFlags(log.Ldate | log.LUTC | log.Ltime | log.Llongfile)

	cfgFile, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load config %q: %w", configPath, err)
	}

	cfg, err := config.ParseConfig(cfgFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse config %q: %w", configPath, err)
	}

	appLogger := logger.NewLogger(&cfg.Logger)
	appLogger.InitLogger()

	logger.SetDefaultLogger(appLogger)

	if cfg.Logger.Development {
		appLogger.Infof("config: %+v", cfg)
	}

	ctx, stop := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)

	return ctx, stop, cfg, nil
}
