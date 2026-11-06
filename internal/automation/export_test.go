package automation

// Match exposes match for testing in package automation_test.
func Match(target, data any) bool {
	return match(target, data)
}

// IsNumeric exposes isNumeric for testing in package automation_test.
func IsNumeric(v any) bool {
	return isNumeric(v)
}

// NumericEqual exposes numericEqual for testing in package automation_test.
func NumericEqual(a, b any) bool {
	return numericEqual(a, b)
}
