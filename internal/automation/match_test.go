package automation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MyChaOS87/automqttion/internal/automation"
)

type matchTestCases []struct {
	name     string
	target   any
	data     any
	expected bool
}

func TestMatch(t *testing.T) {
	t.Parallel()

	tests := matchTestCases{
		{
			name:     "both nil",
			target:   nil,
			data:     nil,
			expected: true,
		},
		{
			name:     "nil target and non-nil data",
			target:   nil,
			data:     "value",
			expected: false,
		},
		{
			name:     "non-nil target and nil data",
			target:   "value",
			data:     nil,
			expected: false,
		},
		{
			name:     "equal ints",
			target:   1,
			data:     1,
			expected: true,
		},
		{
			name:     "unequal ints",
			target:   1,
			data:     0,
			expected: false,
		},
		{
			name:     "negative ints equal",
			target:   -42,
			data:     -42,
			expected: true,
		},
		{
			name:     "int and string mismatch",
			target:   1,
			data:     "test",
			expected: false,
		},
		{
			name:     "equal strings",
			target:   "test",
			data:     "test",
			expected: true,
		},
		{
			name:     "empty strings equal",
			target:   "",
			data:     "",
			expected: true,
		},
		{
			name:     "unequal strings",
			target:   "test",
			data:     "test1",
			expected: false,
		},
		{
			name:     "booleans equal",
			target:   true,
			data:     true,
			expected: true,
		},
		{
			name:     "booleans unequal",
			target:   true,
			data:     false,
			expected: false,
		},
		{
			name:     "slices equal",
			target:   []string{"a", "b"},
			data:     []string{"a", "b"},
			expected: true,
		},
		{
			name:     "slices unequal",
			target:   []string{"a", "b"},
			data:     []string{"a", "c"},
			expected: false,
		},
		{
			name:     "int and float64 equal zero",
			target:   int(0),
			data:     float64(0.0),
			expected: true,
		},
		{
			name:     "float64 and int equal zero",
			target:   float64(0.0),
			data:     int(0),
			expected: true,
		},
		{
			name:     "uint and float64 equal",
			target:   uint(7),
			data:     float64(7.0),
			expected: true,
		},
		{
			name:     "float64 with fraction and int mismatch",
			target:   float64(1.2),
			data:     int(1),
			expected: false,
		},
		{
			name:     "int and float64 with fraction mismatch",
			target:   int(1),
			data:     float64(1.2),
			expected: false,
		},
		{
			name:     "map target against non-map data",
			target:   map[string]any{"a": 1},
			data:     "not-a-map",
			expected: false,
		},
		{
			name:     "exact map match",
			target:   map[string]any{"a": 1},
			data:     map[string]any{"a": 1},
			expected: true,
		},
		{
			name:     "subset map match with extra data keys",
			target:   map[string]any{"a": 1},
			data:     map[string]any{"a": 1, "b": 2},
			expected: true,
		},
		{
			name:     "subset map value mismatch",
			target:   map[string]any{"a": 1},
			data:     map[string]any{"a": 2, "b": 2},
			expected: false,
		},
		{
			name:     "subset map missing key in data",
			target:   map[string]any{"a": 1, "missing": "val"},
			data:     map[string]any{"a": 1},
			expected: false,
		},
		{
			name:     "multi-key subset map match",
			target:   map[string]any{"a": 1, "c": "test"},
			data:     map[string]any{"a": 1, "b": 2, "c": "test"},
			expected: true,
		},
		{
			name:     "deeply nested map match with numeric coercion",
			target:   map[string]any{"a": map[string]any{"c": "test", "count": 5}},
			data:     map[string]any{"a": map[string]any{"c": "test", "count": float64(5.0), "d": 0}, "b": 2},
			expected: true,
		},
		{
			name:     "deeply nested map type mismatch",
			target:   map[string]any{"a": map[string]any{"c": "test"}},
			data:     map[string]any{"a": "not-a-nested-map", "b": 2},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := automation.Match(tt.target, tt.data)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

type isNumericTestCases []struct {
	name     string
	input    any
	expected bool
}

func TestIsNumeric(t *testing.T) {
	t.Parallel()

	tests := isNumericTestCases{
		{name: "int", input: int(1), expected: true},
		{name: "int8", input: int8(1), expected: true},
		{name: "int16", input: int16(1), expected: true},
		{name: "int32", input: int32(1), expected: true},
		{name: "int64", input: int64(1), expected: true},
		{name: "uint", input: uint(1), expected: true},
		{name: "uint8", input: uint8(1), expected: true},
		{name: "uint16", input: uint16(1), expected: true},
		{name: "uint32", input: uint32(1), expected: true},
		{name: "uint64", input: uint64(1), expected: true},
		{name: "float32", input: float32(1.5), expected: true},
		{name: "float64", input: float64(1.5), expected: true},
		{name: "string", input: "1", expected: false},
		{name: "bool", input: true, expected: false},
		{name: "nil", input: nil, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, automation.IsNumeric(tt.input))
		})
	}
}

type numericEqualTestCases []struct {
	name     string
	a        any
	b        any
	expected bool
}

func TestNumericEqual(t *testing.T) {
	t.Parallel()

	tests := numericEqualTestCases{
		{name: "int and int64 equal", a: int(10), b: int64(10), expected: true},
		{name: "uint8 and uint64 equal", a: uint8(255), b: uint64(255), expected: true},
		{name: "int32 and float64 equal", a: int32(-15), b: float64(-15.0), expected: true},
		{name: "float32 and float64 equal", a: float32(2.5), b: float64(2.5), expected: true},
		{name: "float64 truncation mismatch", a: float64(2.75), b: int(2), expected: false},
		{name: "non-numeric first arg", a: "10", b: int(10), expected: false},
		{name: "non-numeric second arg", a: int(10), b: "10", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, automation.NumericEqual(tt.a, tt.b))
		})
	}
}
