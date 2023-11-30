package automation

import (
	"context"
	"reflect"

	"github.com/MyChaOS87/automqttion.git/config"
	"github.com/MyChaOS87/automqttion.git/pkg/broker"
	"github.com/MyChaOS87/automqttion.git/pkg/log"
	"gopkg.in/yaml.v2"
)

type Automation interface {
	Start(ctx context.Context, cancel context.CancelFunc)
}

type automation struct {
	cfg    config.AutomateConfig
	broker broker.Broker
}

func (a *automation) Start(ctx context.Context, cancel context.CancelFunc) {
	matcher := replaceIfYamlMatcher(a.cfg.On.Match)

	a.broker.Topic(a.cfg.On.Topic).
		SubscribeRawJson(func(topic string, data interface{}) {
			if match(matcher, dereference(data)) {
				for _, action := range a.cfg.Do {
					a.broker.Topic(action.Topic).Publish(action.Object)
				}
			}
		})
}

func replaceIfYamlMatcher(m any) any {
	val := reflect.ValueOf(m)
	if val.Kind() != reflect.Map {
		return m
	}
	iter := val.MapRange()
	for iter.Next() {
		k := iter.Key()
		v := iter.Value()

		if k.Kind() != reflect.String {
			return m
		}

		if reflect.DeepEqual(k.Interface(), "$yaml") {
			var newVal map[string]any
			s, ok := v.Interface().(string)
			if !ok {
				log.Fatalf("$yaml element can only hold a string")
			}

			err := yaml.Unmarshal([]byte(s), &newVal)
			if err != nil {
				log.Fatalf("cannot parse configuration §yaml tag: %s", err)
			}

			return newVal
		}
	}

	return m
}

func dereference(a any) any {
	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Pointer {
		return v.Elem().Interface()
	}

	return a
}

func NewAutomation(broker broker.Broker, cfg config.AutomateConfig) Automation {
	return &automation{
		cfg:    cfg,
		broker: broker,
	}
}
