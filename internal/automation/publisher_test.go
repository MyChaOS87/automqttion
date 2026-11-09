package automation_test

import (
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/internal/automation"
	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt/mqtttest"
)

type publisherTestCases []struct {
	name      string
	topic     string
	content   any
	expectErr bool
	expected  *mqtttest.PublishedMessage
}

func TestPublisher(t *testing.T) {
	t.Parallel()

	tests := publisherTestCases{
		{
			name:    "explicit $plain string payload",
			topic:   "zigbee2mqtt/light/set",
			content: map[string]any{"$plain": "TOGGLE"},
			expected: &mqtttest.PublishedMessage{
				Topic:   "zigbee2mqtt/light/set",
				QoS:     1,
				Payload: "TOGGLE",
			},
		},
		{
			name:    "explicit $plain integer payload formatted via Sprintf",
			topic:   "zigbee2mqtt/dimmer/set",
			content: map[string]any{"$plain": 100},
			expected: &mqtttest.PublishedMessage{
				Topic:   "zigbee2mqtt/dimmer/set",
				QoS:     1,
				Payload: "100",
			},
		},
		{
			name:    "explicit $json map payload unwraps inner content",
			topic:   "zigbee2mqtt/livingroom/set",
			content: map[string]any{"$json": map[string]any{"state": "ON", "brightness": 200}},
			expected: &mqtttest.PublishedMessage{
				Topic:   "zigbee2mqtt/livingroom/set",
				QoS:     1,
				Payload: `{"brightness":200,"state":"ON"}`,
			},
		},
		{
			name:    "untagged map falls back to jsonPublisher with full map",
			topic:   "zigbee2mqtt/kitchen/set",
			content: map[string]any{"state": "OFF"},
			expected: &mqtttest.PublishedMessage{
				Topic:   "zigbee2mqtt/kitchen/set",
				QoS:     1,
				Payload: `{"state":"OFF"}`,
			},
		},
		{
			name:    "map with non-string key falls back to jsonPublisher",
			topic:   "zigbee2mqtt/raw/set",
			content: map[int]string{1: "one"},
			expected: &mqtttest.PublishedMessage{
				Topic:   "zigbee2mqtt/raw/set",
				QoS:     1,
				Payload: `{"1":"one"}`,
			},
		},
		{
			name:    "non-map primitive content falls back to jsonPublisher",
			topic:   "zigbee2mqtt/primitive/set",
			content: "direct-value",
			expected: &mqtttest.PublishedMessage{
				Topic:   "zigbee2mqtt/primitive/set",
				QoS:     1,
				Payload: `"direct-value"`,
			},
		},
		{
			name:      "unmarshalable jsonPublisher content returns error",
			topic:     "zigbee2mqtt/invalid/set",
			content:   make(chan int),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				brk, sim := mqtttest.NewBroker()

				pub := automation.NewPublisher(brk.Topic(tt.topic), tt.content)
				require.NotNil(t, pub)

				err := pub.Publish()
				if tt.expectErr {
					require.Error(t, err)
					assert.Empty(t, sim.PublishedMessages())

					return
				}

				require.NoError(t, err)
				synctest.Wait()

				published := sim.PublishedMessages()
				require.Len(t, published, 1)
				assert.Equal(t, *tt.expected, published[0])
			})
		})
	}
}
