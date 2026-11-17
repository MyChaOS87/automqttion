package main

import (
	"fmt"
	stdlog "log"
	"os"

	"github.com/MyChaOS87/automqttion/internal/automation"
	"github.com/MyChaOS87/automqttion/internal/cmd"
	"github.com/MyChaOS87/automqttion/pkg/log"
)

// Version is stamped at build time via -ldflags.
var Version = "dev" //nolint:gochecknoglobals // build-time ldflags version

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel, cfg, err := cmd.Init()
	if err != nil {
		stdlog.Printf("initialization failed: %v", err)

		return fmt.Errorf("init: %w", err)
	}
	defer cancel()

	log.Infof("Starting automqttion version %s", Version)

	broker, err := cmd.NewMQTTBroker(cfg.Mqtt)
	if err != nil {
		log.Errorf("invalid mqtt configuration: %v", err)

		return fmt.Errorf("configure mqtt broker: %w", err)
	}

	automations := make([]automation.Automation, 0, len(cfg.Automate))

	for topic, matchers := range cfg.Automate {
		auto, err := automation.NewAutomation(broker, topic, matchers)
		if err != nil {
			log.Errorf("cannot create automation for topic %s: %v", topic, err)

			return fmt.Errorf("create automation for topic %q: %w", topic, err)
		}

		automations = append(automations, auto)
	}

	for _, auto := range automations {
		auto.Start()
	}

	if err := broker.Run(ctx); err != nil {
		log.Errorf("mqtt broker stopped with error: %v", err)

		return fmt.Errorf("run mqtt broker: %w", err)
	}

	return nil
}
