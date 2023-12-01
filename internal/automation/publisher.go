package automation

import (
	"fmt"
	"reflect"

	"github.com/MyChaOS87/automqttion.git/pkg/broker"
)

type publisher interface {
	Publish()
}

type commonPublisher struct {
	topic broker.Topic
}

func newCommonPublisher(topic broker.Topic) commonPublisher {
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

func (j jsonPublisher) Publish() {
	j.topic.Publish(j.content)
}

type plainPublisher struct {
	commonPublisher
	content any
}

func (p plainPublisher) Publish() {
	p.topic.PublishString(fmt.Sprintf("%v", p.content))
}

func newPublisher(topic broker.Topic, content any) publisher {
	commonPublisher := newCommonPublisher(topic)

	defaultPublisher := func() publisher {
		return &jsonPublisher{
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

		if reflect.DeepEqual(k.Interface(), "$json") {
			return jsonPublisher{
				commonPublisher: commonPublisher,
				content:         v.Interface(),
			}
		} else if reflect.DeepEqual(k.Interface(), "$plain") {
			return plainPublisher{
				commonPublisher: commonPublisher,
				content:         v.Interface(),
			}
		}
	}

	return defaultPublisher()
}
