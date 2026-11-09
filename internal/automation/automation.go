package automation

import (
	"errors"
	"fmt"
	"reflect"

	"go.yaml.in/yaml/v3"

	"github.com/MyChaOS87/automqttion/config"
	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt"
	"github.com/MyChaOS87/automqttion/pkg/log"
)

var (
	ErrYamlMatcherNotString = errors.New("$yaml element can only hold a string")
	ErrYamlMatcherParse     = errors.New("cannot parse configuration $yaml tag")
)

type Automation interface {
	Start()
}

type matcher struct {
	match      any
	publishers []publisher
}

type automation struct {
	topic    string
	matchers []matcher
	broker   *mqtt.Broker
}

func (a *automation) Start() {
	a.broker.Topic(a.topic).
		Subscribe(func(_ string, data any) {
			for _, matcher := range a.matchers {
				if match(matcher.match, dereference(data)) {
					for _, publisher := range matcher.publishers {
						if err := publisher.Publish(); err != nil {
							log.Errorf("cannot publish action: %s", err)
						}
					}
				}
			}
		})
}

func replaceIfYamlMatcher(m any) (any, error) {
	val := reflect.ValueOf(m)
	if val.Kind() != reflect.Map {
		return m, nil
	}

	iter := val.MapRange()

	for iter.Next() {
		k := iter.Key()
		v := iter.Value()

		if k.Kind() != reflect.String {
			return m, nil
		}

		if reflect.DeepEqual(k.Interface(), "$yaml") {
			var newVal map[string]any

			s, ok := v.Interface().(string)
			if !ok {
				return nil, ErrYamlMatcherNotString
			}

			err := yaml.Unmarshal([]byte(s), &newVal)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrYamlMatcherParse, err)
			}

			return newVal, nil
		}
	}

	return m, nil
}

func dereference(a any) any {
	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Pointer {
		return v.Elem().Interface()
	}

	return a
}

func NewAutomation(broker *mqtt.Broker, topic string, matchersConfig []config.MatchConfig) (Automation, error) {
	matchers := make([]matcher, 0, len(matchersConfig))

	for _, cfg := range matchersConfig {
		matched, err := replaceIfYamlMatcher(cfg.Match)
		if err != nil {
			return nil, err
		}

		publishers := make([]publisher, 0, len(cfg.Actions))
		for _, action := range cfg.Actions {
			publishers = append(publishers, newPublisher(broker.Topic(action.Topic), action.Content))
		}

		matchers = append(matchers, matcher{
			match:      matched,
			publishers: publishers,
		})
	}

	return &automation{
		topic:    topic,
		matchers: matchers,
		broker:   broker,
	}, nil
}
