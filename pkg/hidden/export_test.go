package hidden

// ExportedGroup exposes the unexported group struct for testing in package hidden_test.
type ExportedGroup = group

// NewExportedGroup constructs an ExportedGroup with the given start and end indices.
func NewExportedGroup(start, end int) ExportedGroup {
	return group{start: start, end: end}
}

// UnifyGroups exposes unifyGroups for testing in package hidden_test.
func UnifyGroups(groups []ExportedGroup) []ExportedGroup {
	return unifyGroups(groups)
}

// SortGroups exposes sortGroups for testing in package hidden_test.
func SortGroups(a, b ExportedGroup) int {
	return sortGroups(a, b)
}
