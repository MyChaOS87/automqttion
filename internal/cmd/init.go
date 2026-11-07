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

func Init() (ctx context.Context, cancel context.CancelFunc, cfg *config.Config, err error) {
	log.Default().SetFlags(log.Ldate | log.LUTC | log.Ltime | log.Llongfile)

	cfgFile, err := config.LoadConfig(configFilename)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load config (%s): %w", configFilename, err)
	}

	cfg, err = config.ParseConfig(cfgFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse config: %w", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	ctx, cancel = context.WithCancel(context.Background())

	appLogger := logger.NewLogger(&cfg.Logger)
	appLogger.InitLogger()

	logger.SetDefaultLogger(appLogger)

	if cfg.Logger.Development {
		appLogger.Infof("config: %+v", cfg)
	}

	go func() {
		<-quit

		appLogger.Info("killed by signal")
		cancel()
	}()

	return ctx, cancel, cfg, nil
}
