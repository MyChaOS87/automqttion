package automation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/internal/automation"
	"github.com/MyChaOS87/automqttion/pkg/broker"
)

type recordedPublish struct {
	topic   string
	isPlain bool
	plain   string
	jsonVal any
}

type recordingTopic struct {
	name          string
	published     *[]recordedPublish
	rawJSONSub    broker.CallbackFunction
	typedSub      broker.CallbackFunction
	subscribedHnt any
}

func (rt *recordingTopic) PublishString(s string) {
	*rt.published = append(*rt.published, recordedPublish{
		topic:   rt.name,
		isPlain: true,
		plain:   s,
	})
}

func (rt *recordingTopic) Publish(i any) {
	*rt.published = append(*rt.published, recordedPublish{
		topic:   rt.name,
		isPlain: false,
		jsonVal: i,
	})
}

func (rt *recordingTopic) Subscribe(hint any, callback broker.CallbackFunction) {
	rt.subscribedHnt = hint
	rt.typedSub = callback
}

func (rt *recordingTopic) SubscribeRawJSON(callback broker.CallbackFunction) {
	rt.rawJSONSub = callback
}

type publisherTestCases []struct {
	name     string
	topic    string
	content  any
	expected recordedPublish
}

func TestPublisher(t *testing.T) {
	t.Parallel()

	tests := publisherTestCases{
		{
			name:    "explicit $plain string payload",
			topic:   "zigbee2mqtt/light/set",
			content: map[string]any{"$plain": "TOGGLE"},
			expected: recordedPublish{
				topic:   "zigbee2mqtt/light/set",
				isPlain: true,
				plain:   "TOGGLE",
			},
		},
		{
			name:    "explicit $plain integer payload formatted via Sprintf",
			topic:   "zigbee2mqtt/dimmer/set",
			content: map[string]any{"$plain": 100},
			expected: recordedPublish{
				topic:   "zigbee2mqtt/dimmer/set",
				isPlain: true,
				plain:   "100",
			},
		},
		{
			name:    "explicit $json map payload unwraps inner content",
			topic:   "zigbee2mqtt/livingroom/set",
			content: map[string]any{"$json": map[string]any{"state": "ON", "brightness": 200}},
			expected: recordedPublish{
				topic:   "zigbee2mqtt/livingroom/set",
				isPlain: false,
				jsonVal: map[string]any{"state": "ON", "brightness": 200},
			},
		},
		{
			name:    "untagged map falls back to jsonPublisher with full map",
			topic:   "zigbee2mqtt/kitchen/set",
			content: map[string]any{"state": "OFF"},
			expected: recordedPublish{
				topic:   "zigbee2mqtt/kitchen/set",
				isPlain: false,
				jsonVal: map[string]any{"state": "OFF"},
			},
		},
		{
			name:    "map with non-string key falls back to jsonPublisher",
			topic:   "zigbee2mqtt/raw/set",
			content: map[int]string{1: "one"},
			expected: recordedPublish{
				topic:   "zigbee2mqtt/raw/set",
				isPlain: false,
				jsonVal: map[int]string{1: "one"},
			},
		},
		{
			name:    "non-map primitive content falls back to jsonPublisher",
			topic:   "zigbee2mqtt/primitive/set",
			content: "direct-value",
			expected: recordedPublish{
				topic:   "zigbee2mqtt/primitive/set",
				isPlain: false,
				jsonVal: "direct-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var published []recordedPublish

			topic := &recordingTopic{
				name:      tt.topic,
				published: &published,
			}

			pub := automation.NewPublisher(topic, tt.content)
			require.NotNil(t, pub)

			pub.Publish()

			require.Len(t, published, 1)
			assert.Equal(t, tt.expected, published[0])
		})
	}
}
