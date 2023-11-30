package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MyChaOS87/automqttion.git/config"
	logger "github.com/MyChaOS87/automqttion.git/pkg/log"
)

const (
	configFilename = "./config/config"
)

func Init() (ctx context.Context, cancel context.CancelFunc, cfg *config.Config) {
	log.Default().SetFlags(log.Ldate | log.LUTC | log.Ltime | log.Llongfile)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	ctx, cancel = context.WithCancel(context.Background())

	cfgFile, err := config.LoadConfig(configFilename)
	if err != nil {
		log.Printf("FATAL: LoadConfig(%v): %v", configFilename, err)
		cancel()
	}

	cfg, err = config.ParseConfig(cfgFile)
	if err != nil {
		log.Printf("FATAL: ParseConfig: %v", err)
		cancel()
	}

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

	return ctx, cancel, cfg
}
