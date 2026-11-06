package main

import (
	"github.com/MyChaOS87/automqttion/internal/automation"
	"github.com/MyChaOS87/automqttion/internal/cmd"
	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt"
	"github.com/MyChaOS87/automqttion/pkg/log"
)

func main() {
	ctx, cancel, cfg := cmd.Init()
	defer cancel()

	broker := mqtt.NewBroker(
		mqtt.Broker(cfg.Mqtt.Broker))

	broker.Run(ctx, cancel)

	for topic, matchers := range cfg.Automate {
		automation.NewAutomation(broker, topic, matchers).Start(ctx, cancel)
	}

	<-ctx.Done()

	log.Errorf("context done: %s", ctx.Err().Error())
}
