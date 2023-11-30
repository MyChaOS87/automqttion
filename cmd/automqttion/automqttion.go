package main

import (
	"github.com/MyChaOS87/automqttion.git/internal/automation"
	"github.com/MyChaOS87/automqttion.git/internal/cmd"
	"github.com/MyChaOS87/automqttion.git/pkg/broker/mqtt"
	"github.com/MyChaOS87/automqttion.git/pkg/log"
)

func main() {
	ctx, cancel, cfg := cmd.Init()
	defer cancel()

	broker := mqtt.NewBroker(
		mqtt.Broker(cfg.Mqtt.Broker))

	broker.Run(ctx, cancel)

	for _, a := range cfg.Automate {
		automation.NewAutomation(broker, a).Start(ctx, cancel)
	}

	<-ctx.Done()

	log.Errorf("context done: %s", ctx.Err().Error())
}
