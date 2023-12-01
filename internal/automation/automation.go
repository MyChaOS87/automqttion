package automation

import (
	"context"
	"reflect"

	"gopkg.in/yaml.v2"

	"github.com/MyChaOS87/automqttion.git/config"
	"github.com/MyChaOS87/automqttion.git/pkg/broker"
	"github.com/MyChaOS87/automqttion.git/pkg/log"
)

type Automation interface {
	Start(ctx context.Context, cancel context.CancelFunc)
}

type matcher struct {
	match      any
	publishers []publisher
}

type automation struct {
	topic    string
	matchers []matcher
	broker   broker.Broker
}

func (a *automation) Start(_ context.Context, _ context.CancelFunc) {
	a.broker.Topic(a.topic).
		SubscribeRawJSON(func(topic string, data interface{}) {
			for _, matcher := range a.matchers {
				if match(matcher.match, dereference(data)) {
					for _, publisher := range matcher.publishers {
						publisher.Publish()
					}
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

func NewAutomation(broker broker.Broker, topic string, matchersConfig []config.MatchConfig) Automation {
	matchers := make([]matcher, 0, len(matchersConfig))

	for _, cfg := range matchersConfig {
		publishers := make([]publisher, 0, len(cfg.Actions))
		for _, action := range cfg.Actions {
			publishers = append(publishers, newPublisher(broker.Topic(action.Topic), action.Content))
		}

		matchers = append(matchers, matcher{
			match:      replaceIfYamlMatcher(cfg.Match),
			publishers: publishers,
		})
	}

	return &automation{
		topic:    topic,
		matchers: matchers,
		broker:   broker,
	}
}
