package hidden

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// SubstringSelector provides a list of regular expressions used to select parts
// of a string that should be masked.
type SubstringSelector interface {
	// Selectors returns the regular expressions to match. If a regex contains no capture groups,
	// the entire match is masked. If it contains capture groups, only the captured submatches are masked.
	Selectors() []*regexp.Regexp
}

// Substring represents a string where only specific segments (defined by the [SubstringSelector])
// are masked when converted to a string or serialized to JSON.
//
// Use Reveal() to retrieve the raw unmasked string.
type Substring[Selector SubstringSelector] string

// String returns the masked string with matches replaced by the placeholder "*HIDDEN*".
func (s Substring[T]) String() string {
	var t T

	selectors := t.Selectors()

	res := string(s)

	for _, selector := range selectors {
		if selector == nil {
			continue
		}

		if selector.NumSubexp() == 0 {
			res = selector.ReplaceAllString(res, HiddenText)
		} else {
			res = replaceCaptureGroups(selector, res, HiddenText)
		}
	}

	return res
}

// GoString masks matched segments when formatted with %#v.
func (s Substring[T]) GoString() string {
	return strconv.Quote(s.String())
}

// Reveal returns the raw, unmasked string value.
func (s Substring[T]) Reveal() string {
	return string(s)
}

// MarshalJSON serializes the Substring into a JSON string containing the masked value.
func (s Substring[T]) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(s.String())), nil
}

// replaceCaptureGroups replaces the capture groups in the input string with the replacement string.
// It uses the provided regular expression to find the capture groups and replaces them accordingly.
func replaceCaptureGroups(re *regexp.Regexp, input, replacement string) string {
	cgs := captureGroups(re, input)
	cgs = unifyGroups(cgs)

	unhiddenInput := make([]string, 0, len(cgs)+1)
	start := 0

	for _, group := range cgs {
		unhiddenInput = append(unhiddenInput, input[start:group.start])
		start = group.end
	}

	unhiddenInput = append(unhiddenInput, input[start:])

	return strings.Join(unhiddenInput, replacement)
}

type group struct {
	start int
	end   int
}

func captureGroups(re *regexp.Regexp, input string) []group {
	matchIndexes := re.FindAllStringSubmatchIndex(input, -1)

	groups := make([]group, 0)

	for _, match := range matchIndexes {
		for i := 2; i < len(match)-1; i += 2 {
			start := match[i]
			end := match[i+1]

			if end-start > 0 {
				groups = append(groups, group{start: start, end: end})
			}
		}
	}

	return groups
}

func unifyGroups(groups []group) []group {
	if len(groups) < 2 {
		return groups
	}

	slices.SortFunc(groups, sortGroups)

	unified := make([]group, 0, len(groups))
	unified = append(unified, groups[0])

	for _, g := range groups[1:] {
		lastIdx := len(unified) - 1

		if g.start <= unified[lastIdx].end {
			unified[lastIdx].end = max(unified[lastIdx].end, g.end)
		} else {
			unified = append(unified, g)
		}
	}

	return unified
}

func sortGroups(a, b group) int {
	switch {
	case a.start < b.start:
		return -1
	case a.start > b.start:
		return 1
	case a.end < b.end:
		return -1
	case a.end > b.end:
		return 1
	default:
		return 0
	}
}
