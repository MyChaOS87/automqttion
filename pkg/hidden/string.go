package hidden

import "strconv"

// String represents a sensitive string value that is completely masked when converted
// to a string (e.g. for logging) or serialized to JSON.
//
// Use Reveal() to access the raw underlying secret value. This type is ideal for standalone
// credentials, API keys, or private signatures.
type String string

// String returns the masked text placeholder ("*HIDDEN*"). If the secret is empty, it returns "".
func (s String) String() string {
	if s == "" {
		return ""
	}

	return HiddenText
}

// GoString masks the value when formatted with %#v.
func (s String) GoString() string {
	return strconv.Quote(s.String())
}

// Reveal returns the raw, unmasked string value. Use this when passing the secret
// to external clients or resource initializers.
func (s String) Reveal() string {
	return string(s)
}

// MarshalJSON serializes the String into a JSON string containing the masked placeholder.
func (s String) MarshalJSON() ([]byte, error) {
	if s == "" {
		return []byte(`""`), nil
	}

	return []byte(strconv.Quote(HiddenText)), nil
}
