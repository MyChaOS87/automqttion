package hidden_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MyChaOS87/automqttion/pkg/hidden"
)

func TestUnifyGroups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		groups   []hidden.ExportedGroup
		expected []hidden.ExportedGroup
	}{
		{
			name:     "empty groups",
			groups:   []hidden.ExportedGroup{},
			expected: []hidden.ExportedGroup{},
		},
		{
			name: "single group",
			groups: []hidden.ExportedGroup{
				hidden.NewExportedGroup(0, 5),
			},
			expected: []hidden.ExportedGroup{hidden.NewExportedGroup(0, 5)},
		},
		{
			name: "overlapping groups",
			groups: []hidden.ExportedGroup{
				hidden.NewExportedGroup(0, 5),
				hidden.NewExportedGroup(3, 8),
			},
			expected: []hidden.ExportedGroup{hidden.NewExportedGroup(0, 8)},
		},
		{
			name: "non-overlapping groups",
			groups: []hidden.ExportedGroup{
				hidden.NewExportedGroup(0, 5),
				hidden.NewExportedGroup(6, 10),
			},
			expected: []hidden.ExportedGroup{hidden.NewExportedGroup(0, 5), hidden.NewExportedGroup(6, 10)},
		},
		{
			name: "complex overlapping groups",
			groups: []hidden.ExportedGroup{
				hidden.NewExportedGroup(0, 5),
				hidden.NewExportedGroup(4, 10),
				hidden.NewExportedGroup(8, 12),
			},
			expected: []hidden.ExportedGroup{hidden.NewExportedGroup(0, 12)},
		},
		{
			name: "nested groups",
			groups: []hidden.ExportedGroup{
				hidden.NewExportedGroup(0, 10),
				hidden.NewExportedGroup(2, 5),
				hidden.NewExportedGroup(6, 8),
			},
			expected: []hidden.ExportedGroup{hidden.NewExportedGroup(0, 10)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := hidden.UnifyGroups(tt.groups)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortGroups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		groups   []hidden.ExportedGroup
		expected []hidden.ExportedGroup
	}{
		{
			name:     "empty groups",
			groups:   []hidden.ExportedGroup{},
			expected: []hidden.ExportedGroup{},
		},
		{
			name: "single group",
			groups: []hidden.ExportedGroup{
				hidden.NewExportedGroup(0, 5),
			},
			expected: []hidden.ExportedGroup{hidden.NewExportedGroup(0, 5)},
		},
		{
			name: "multiple groups",
			groups: []hidden.ExportedGroup{
				hidden.NewExportedGroup(5, 10),
				hidden.NewExportedGroup(0, 3),
				hidden.NewExportedGroup(2, 4),
			},
			expected: []hidden.ExportedGroup{
				hidden.NewExportedGroup(0, 3),
				hidden.NewExportedGroup(2, 4),
				hidden.NewExportedGroup(5, 10),
			},
		},
		{
			name: "groups with same start",
			groups: []hidden.ExportedGroup{
				hidden.NewExportedGroup(2, 5),
				hidden.NewExportedGroup(2, 4),
				hidden.NewExportedGroup(2, 6),
			},
			expected: []hidden.ExportedGroup{
				hidden.NewExportedGroup(2, 4),
				hidden.NewExportedGroup(2, 5),
				hidden.NewExportedGroup(2, 6),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			slices.SortFunc(tt.groups, hidden.SortGroups)
			assert.Equal(t, tt.expected, tt.groups)
		})
	}
}
