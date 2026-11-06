package automation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/config"
	"github.com/MyChaOS87/automqttion/internal/automation"
	"github.com/MyChaOS87/automqttion/pkg/broker"
)

type recordingBroker struct {
	topics    map[string]*recordingTopic
	published []recordedPublish
}

func newRecordingBroker() *recordingBroker {
	return &recordingBroker{
		topics: make(map[string]*recordingTopic),
	}
}

func (rb *recordingBroker) Run(_ context.Context, _ context.CancelFunc) {}

func (rb *recordingBroker) Topic(name string) broker.Topic {
	if t, ok := rb.topics[name]; ok {
		return t
	}

	t := &recordingTopic{
		name:      name,
		published: &rb.published,
	}
	rb.topics[name] = t

	return t
}

type replaceIfYamlMatcherTestCases []struct {
	name     string
	input    any
	expected any
}

func TestReplaceIfYamlMatcher(t *testing.T) {
	t.Parallel()

	tests := replaceIfYamlMatcherTestCases{
		{
			name:     "non-map string returned unchanged",
			input:    "simple-string",
			expected: "simple-string",
		},
		{
			name:     "non-map int returned unchanged",
			input:    42,
			expected: 42,
		},
		{
			name:     "map with non-string key returned unchanged",
			input:    map[int]string{1: "val"},
			expected: map[int]string{1: "val"},
		},
		{
			name:     "standard map without $yaml key returned unchanged",
			input:    map[string]any{"action": "single", "battery": 100},
			expected: map[string]any{"action": "single", "battery": 100},
		},
		{
			name:  "map with $yaml key expands YAML into map",
			input: map[string]any{"$yaml": "action: single\nbrightness: 128\n"},
			expected: map[string]any{
				"action":     "single",
				"brightness": 128,
			},
		},
		{
			name:  "map with nested $yaml document expands nested map",
			input: map[string]any{"$yaml": "device:\n  state: ON\n  level: 5\n"},
			expected: map[string]any{
				"device": map[string]any{
					"state": "ON",
					"level": 5,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := automation.ReplaceIfYamlMatcher(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

type dereferenceTestCases []struct {
	name     string
	input    any
	expected any
}

func TestDereference(t *testing.T) {
	t.Parallel()

	strVal := "hello"
	intVal := 99
	mapVal := map[string]any{"action": "toggle"}

	tests := dereferenceTestCases{
		{
			name:     "non-pointer string",
			input:    "direct",
			expected: "direct",
		},
		{
			name:     "non-pointer map",
			input:    map[string]any{"k": "v"},
			expected: map[string]any{"k": "v"},
		},
		{
			name:     "pointer to string dereferenced",
			input:    &strVal,
			expected: "hello",
		},
		{
			name:     "pointer to int dereferenced",
			input:    &intVal,
			expected: 99,
		},
		{
			name:     "pointer to map dereferenced",
			input:    &mapVal,
			expected: map[string]any{"action": "toggle"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := automation.Dereference(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

type automationStartTestCases []struct {
	name          string
	sourceTopic   string
	matchConfigs  []config.MatchConfig
	inboundEvents []any
	expectedPubs  []recordedPublish
}

func TestAutomationStart(t *testing.T) {
	t.Parallel()

	ptrPayload := any(map[string]any{
		"action":      "single",
		"linkquality": float64(120),
	})

	tests := automationStartTestCases{
		{
			name:        "matching pointer payload dispatches $plain and $json actions",
			sourceTopic: "zigbee2mqtt/switch_1",
			matchConfigs: []config.MatchConfig{
				{
					Match: map[string]any{"action": "single"},
					Actions: []config.ActionConfig{
						{
							Topic:   "zigbee2mqtt/light_1/set",
							Content: map[string]any{"$plain": "TOGGLE"},
						},
						{
							Topic:   "zigbee2mqtt/light_2/set",
							Content: map[string]any{"$json": map[string]any{"state": "ON"}},
						},
					},
				},
			},
			inboundEvents: []any{&ptrPayload},
			expectedPubs: []recordedPublish{
				{
					topic:   "zigbee2mqtt/light_1/set",
					isPlain: true,
					plain:   "TOGGLE",
				},
				{
					topic:   "zigbee2mqtt/light_2/set",
					isPlain: false,
					jsonVal: map[string]any{"state": "ON"},
				},
			},
		},
		{
			name:        "$yaml matcher expands and dispatches default JSON action",
			sourceTopic: "zigbee2mqtt/sensor_1",
			matchConfigs: []config.MatchConfig{
				{
					Match: map[string]any{"$yaml": "occupancy: true\niluminance: 10\n"},
					Actions: []config.ActionConfig{
						{
							Topic:   "zigbee2mqtt/hallway/set",
							Content: map[string]any{"state": "ON"},
						},
					},
				},
			},
			inboundEvents: []any{
				map[string]any{
					"occupancy":  true,
					"iluminance": float64(10.0),
					"battery":    95,
				},
			},
			expectedPubs: []recordedPublish{
				{
					topic:   "zigbee2mqtt/hallway/set",
					isPlain: false,
					jsonVal: map[string]any{"state": "ON"},
				},
			},
		},
		{
			name:        "non-matching payload publishes nothing",
			sourceTopic: "zigbee2mqtt/switch_2",
			matchConfigs: []config.MatchConfig{
				{
					Match: map[string]any{"action": "double"},
					Actions: []config.ActionConfig{
						{
							Topic:   "zigbee2mqtt/light_1/set",
							Content: map[string]any{"$plain": "OFF"},
						},
					},
				},
			},
			inboundEvents: []any{
				map[string]any{"action": "single"},
			},
			expectedPubs: nil,
		},
		{
			name:        "multiple matchers dispatch only matching rules across multiple inbound events",
			sourceTopic: "zigbee2mqtt/remote",
			matchConfigs: []config.MatchConfig{
				{
					Match: map[string]any{"action": "on"},
					Actions: []config.ActionConfig{
						{
							Topic:   "zigbee2mqtt/lamp/set",
							Content: map[string]any{"$plain": "ON"},
						},
					},
				},
				{
					Match: map[string]any{"action": "off"},
					Actions: []config.ActionConfig{
						{
							Topic:   "zigbee2mqtt/lamp/set",
							Content: map[string]any{"$plain": "OFF"},
						},
					},
				},
			},
			inboundEvents: []any{
				map[string]any{"action": "off"},
				map[string]any{"action": "unknown"},
				map[string]any{"action": "on"},
			},
			expectedPubs: []recordedPublish{
				{
					topic:   "zigbee2mqtt/lamp/set",
					isPlain: true,
					plain:   "OFF",
				},
				{
					topic:   "zigbee2mqtt/lamp/set",
					isPlain: true,
					plain:   "ON",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			brk := newRecordingBroker()
			auto := automation.NewAutomation(brk, tt.sourceTopic, tt.matchConfigs)
			require.NotNil(t, auto)

			auto.Start(ctx, cancel)

			srcTopic, exists := brk.topics[tt.sourceTopic]
			require.True(t, exists)
			require.NotNil(t, srcTopic.rawJSONSub)

			for _, ev := range tt.inboundEvents {
				srcTopic.rawJSONSub(tt.sourceTopic, ev)
			}

			assert.Equal(t, tt.expectedPubs, brk.published)
		})
	}
}
