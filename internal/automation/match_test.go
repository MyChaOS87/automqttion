//nolint:testpackage
package automation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_match(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target any
		data   any
		want   bool
	}{
		{"nil", nil, nil, true},
		{"1-1", 1, 1, true},
		{"1-0", 1, 0, false},
		{"int-string", 1, "test", false},
		{"string-string", "test", "test", true},
		{"string-string no match", "test", "test1", false},
		{"int-f64", int(0), float64(0.0), true},
		{"f64-int", float64(0.0), int(0), true},
		{"f64-int no match", float64(1.2), int(1), false},
		{"int-f64 no match", int(1), float64(1.2), false},
		{"map-map", map[string]any{"a": 1}, map[string]any{"a": 1}, true},
		{"map-map+x", map[string]any{"a": 1}, map[string]any{"a": 1, "b": 2}, true},
		{"map-map+x no match", map[string]any{"a": 1}, map[string]any{"a": 2, "b": 2}, false},
		{"map1-map+x", map[string]any{"a": 1, "c": "test"}, map[string]any{"a": 1, "b": 2, "c": "test"}, true},
		{
			"deepmap",
			map[string]any{"a": map[string]any{"c": "test"}},
			map[string]any{"a": map[string]any{"c": "test", "d": 0}, "b": 2},
			true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, match(tt.target, tt.data))
		})
	}
}
