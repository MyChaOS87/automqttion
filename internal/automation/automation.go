package automation

import (
	"encoding/json"
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
	ErrPlainMatcherInvalid  = errors.New("$plain element can only hold a scalar value")
)

type Automation interface {
	Start()
}

type matcher struct {
	match      any
	plain      bool
	publishers []publisher
}

type automation struct {
	topic    string
	matchers []matcher
	broker   *mqtt.Broker
}

func (a *automation) Start() {
	a.broker.Topic(a.topic).
		SubscribeRaw(a.handleMessage)
}

func (a *automation) handleMessage(_ string, raw []byte) {
	var (
		jsonParsed  bool
		jsonPayload any
		jsonErr     error
	)

	rawStr := string(raw)

	for _, m := range a.matchers {
		var matched bool

		if m.plain {
			matched = m.match == rawStr
		} else {
			if !jsonParsed {
				jsonErr = json.Unmarshal(raw, &jsonPayload)
				jsonParsed = true
			}

			if jsonErr == nil {
				matched = match(m.match, dereference(jsonPayload))
			} else {
				matched = match(m.match, rawStr)
			}
		}

		if matched {
			for _, p := range m.publishers {
				if err := p.Publish(); err != nil {
					log.Errorf("cannot publish action: %s", err)
				}
			}
		}
	}
}

func compileMatcher(m any) (any, bool, error) {
	val := reflect.ValueOf(m)
	if val.Kind() != reflect.Map {
		return m, false, nil
	}

	iter := val.MapRange()

	for iter.Next() {
		k := iter.Key()
		v := iter.Value()

		if k.Kind() != reflect.String {
			return m, false, nil
		}

		switch k.String() {
		case "$yaml":
			var newVal any

			s, ok := v.Interface().(string)
			if !ok {
				return nil, false, ErrYamlMatcherNotString
			}

			if err := yaml.Unmarshal([]byte(s), &newVal); err != nil {
				return nil, false, fmt.Errorf("%w: %w", ErrYamlMatcherParse, err)
			}

			return newVal, false, nil
		case "$plain":
			raw := v.Interface()
			if !isScalar(raw) {
				return nil, false, ErrPlainMatcherInvalid
			}

			return fmt.Sprintf("%v", raw), true, nil
		}
	}

	return m, false, nil
}

func isScalar(v any) bool {
	switch v.(type) {
	case string, bool:
		return true
	default:
		return isNumeric(v)
	}
}

func replaceIfYamlMatcher(m any) (any, error) {
	compiled, _, err := compileMatcher(m)

	return compiled, err
}

func dereference(a any) any {
	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		return v.Elem().Interface()
	}

	return a
}

func NewAutomation(broker *mqtt.Broker, topic string, matchersConfig []config.MatchConfig) (Automation, error) {
	matchers := make([]matcher, 0, len(matchersConfig))

	for _, cfg := range matchersConfig {
		matched, plain, err := compileMatcher(cfg.Match)
		if err != nil {
			return nil, err
		}

		publishers := make([]publisher, 0, len(cfg.Actions))
		for _, action := range cfg.Actions {
			publishers = append(publishers, newPublisher(broker.Topic(action.Topic), action.Content))
		}

		matchers = append(matchers, matcher{
			match:      matched,
			plain:      plain,
			publishers: publishers,
		})
	}

	return &automation{
		topic:    topic,
		matchers: matchers,
		broker:   broker,
	}, nil
}
