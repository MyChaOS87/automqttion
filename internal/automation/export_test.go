package automation

import "github.com/MyChaOS87/automqttion/pkg/broker"

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

// ExportedPublisher exposes the internal publisher interface for testing in package automation_test.
type ExportedPublisher = publisher

// NewPublisher exposes newPublisher for testing in package automation_test.
func NewPublisher(topic broker.Topic, content any) ExportedPublisher {
	return newPublisher(topic, content)
}

// ReplaceIfYamlMatcher exposes replaceIfYamlMatcher for testing in package automation_test.
func ReplaceIfYamlMatcher(m any) (any, error) {
	return replaceIfYamlMatcher(m)
}

// Dereference exposes dereference for testing in package automation_test.
func Dereference(a any) any {
	return dereference(a)
}
