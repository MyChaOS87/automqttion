package main

import (
	stdlog "log"

	"github.com/MyChaOS87/automqttion/internal/automation"
	"github.com/MyChaOS87/automqttion/internal/cmd"
	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt"
	"github.com/MyChaOS87/automqttion/pkg/log"
)

func main() {
	ctx, cancel, cfg, err := cmd.Init()
	if err != nil {
		stdlog.Fatalf("initialization failed: %v", err)
	}
	defer cancel()

	broker := mqtt.NewBroker(
		mqtt.Broker(cfg.Mqtt.Broker))

	broker.Run(ctx, cancel)

	for topic, matchers := range cfg.Automate {
		auto, err := automation.NewAutomation(broker, topic, matchers)
		if err != nil {
			log.Errorf("cannot create automation for topic %s: %v", topic, err)
			cancel()

			break
		}

		auto.Start(ctx, cancel)
	}

	<-ctx.Done()

	log.Errorf("context done: %s", ctx.Err().Error())
}
