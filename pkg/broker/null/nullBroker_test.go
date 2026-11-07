package null_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/pkg/broker/null"
)

type nullBrokerTestCases []struct {
	name      string
	topicName string
	strMsg    string
	anyMsg    any
}

func TestNullBroker(t *testing.T) {
	t.Parallel()

	tests := nullBrokerTestCases{
		{
			name:      "no-op operations on standard topic execute cleanly",
			topicName: "home/livingroom/light/set",
			strMsg:    "on",
			anyMsg:    map[string]any{"state": "ON"},
		},
		{
			name:      "no-op operations on empty topic execute cleanly",
			topicName: "",
			strMsg:    "",
			anyMsg:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			brk := null.NewBroker()
			require.NotNil(t, brk)

			brk.Run(ctx, cancel)

			topic := brk.Topic(tt.topicName)
			require.NotNil(t, topic)

			assert.NotPanics(t, func() {
				topic.PublishString(tt.strMsg)
				topic.Publish(tt.anyMsg)
				topic.Subscribe(map[string]any{}, func(string, any) {})
				topic.SubscribeRawJSON(func(string, any) {})
			})
		})
	}
}
