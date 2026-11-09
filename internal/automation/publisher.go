package automation

import (
	"fmt"
	"reflect"

	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt"
)

type publisher interface {
	Publish() error
}

type commonPublisher struct {
	topic *mqtt.Topic
}

func newCommonPublisher(topic *mqtt.Topic) commonPublisher {
	return commonPublisher{
		topic: topic,
	}
}

var (
	_ publisher = jsonPublisher{}
	_ publisher = plainPublisher{}
)

type jsonPublisher struct {
	commonPublisher

	content any
}

func (j jsonPublisher) Publish() error {
	if err := j.topic.Publish(j.content); err != nil {
		return fmt.Errorf("publish json content: %w", err)
	}

	return nil
}

type plainPublisher struct {
	commonPublisher

	content any
}

func (p plainPublisher) Publish() error {
	p.topic.PublishString(fmt.Sprintf("%v", p.content))

	return nil
}

func newPublisher(topic *mqtt.Topic, content any) publisher {
	commonPublisher := newCommonPublisher(topic)

	defaultPublisher := func() publisher {
		return jsonPublisher{
			commonPublisher: commonPublisher,
			content:         content,
		}
	}

	val := reflect.ValueOf(content)
	if val.Kind() != reflect.Map {
		return defaultPublisher()
	}

	iter := val.MapRange()
	for iter.Next() {
		k := iter.Key()
		v := iter.Value()

		if k.Kind() != reflect.String {
			return defaultPublisher()
		}

		switch k.String() {
		case "$json":
			return jsonPublisher{
				commonPublisher: commonPublisher,
				content:         v.Interface(),
			}
		case "$plain":
			return plainPublisher{
				commonPublisher: commonPublisher,
				content:         v.Interface(),
			}
		}
	}

	return defaultPublisher()
}
