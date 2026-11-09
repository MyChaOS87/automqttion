package automation_test

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/config"
	"github.com/MyChaOS87/automqttion/internal/automation"
	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt/mqtttest"
)

type replaceIfYamlMatcherTestCases []struct {
	name        string
	input       any
	expected    any
	expectedErr error
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
		{
			name:        "$yaml with non-string value returns ErrYamlMatcherNotString",
			input:       map[string]any{"$yaml": 123},
			expectedErr: automation.ErrYamlMatcherNotString,
		},
		{
			name:        "$yaml with invalid YAML syntax returns ErrYamlMatcherParse",
			input:       map[string]any{"$yaml": "invalid: [yaml"},
			expectedErr: automation.ErrYamlMatcherParse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := automation.ReplaceIfYamlMatcher(tt.input)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				assert.Nil(t, actual)

				return
			}

			require.NoError(t, err)
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
	inboundEvents []string
	expectedPubs  []mqtttest.PublishedMessage
}

func TestAutomationStart(t *testing.T) {
	t.Parallel()

	tests := automationStartTestCases{
		{
			name:        "matching payload dispatches $plain and $json actions",
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
							Topic:   "zigbee2mqtt/light_invalid/set",
							Content: make(chan int),
						},
						{
							Topic:   "zigbee2mqtt/light_2/set",
							Content: map[string]any{"$json": map[string]any{"state": "ON"}},
						},
					},
				},
			},
			inboundEvents: []string{`{"action":"single","linkquality":120}`},
			expectedPubs: []mqtttest.PublishedMessage{
				{
					Topic:   "zigbee2mqtt/light_1/set",
					QoS:     1,
					Payload: "TOGGLE",
				},
				{
					Topic:   "zigbee2mqtt/light_2/set",
					QoS:     1,
					Payload: `{"state":"ON"}`,
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
			inboundEvents: []string{
				`{"occupancy":true,"iluminance":10,"battery":95}`,
			},
			expectedPubs: []mqtttest.PublishedMessage{
				{
					Topic:   "zigbee2mqtt/hallway/set",
					QoS:     1,
					Payload: `{"state":"ON"}`,
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
			inboundEvents: []string{
				`{"action":"single"}`,
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
			inboundEvents: []string{
				`{"action":"off"}`,
				`{"action":"unknown"}`,
				`{"action":"on"}`,
			},
			expectedPubs: []mqtttest.PublishedMessage{
				{
					Topic:   "zigbee2mqtt/lamp/set",
					QoS:     1,
					Payload: "OFF",
				},
				{
					Topic:   "zigbee2mqtt/lamp/set",
					QoS:     1,
					Payload: "ON",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				brk, client := mqtttest.NewBroker()

				auto, err := automation.NewAutomation(brk, tt.sourceTopic, tt.matchConfigs)
				require.NoError(t, err)
				require.NotNil(t, auto)

				auto.Start()

				runErrCh := make(chan error, 1)

				go func() {
					runErrCh <- brk.Run(ctx)
				}()

				synctest.Wait()

				for _, ev := range tt.inboundEvents {
					require.True(t, client.Deliver(tt.sourceTopic, ev))
				}

				synctest.Wait()

				assert.Equal(t, tt.expectedPubs, client.PublishedMessages())

				cancel()
				require.NoError(t, <-runErrCh)
			})
		})
	}
}

func TestNewAutomationError(t *testing.T) {
	t.Parallel()

	brk, _ := mqtttest.NewBroker()
	auto, err := automation.NewAutomation(brk, "zigbee2mqtt/switch", []config.MatchConfig{
		{
			Match: map[string]any{"$yaml": 42},
		},
	})

	require.ErrorIs(t, err, automation.ErrYamlMatcherNotString)
	assert.Nil(t, auto)
}
