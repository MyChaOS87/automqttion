package hidden

import "regexp"

// URLPasswordSelector selects the password portion of a connection URL for masking.
type URLPasswordSelector struct{}

// Selectors returns the regular expression pattern designed to match URL passwords.
func (URLPasswordSelector) Selectors() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(URLPasswordRegexp),
	}
}

// URL represents a string containing a network connection URL (e.g. "tcp://user:password@host").
// It automatically masks the password section while leaving the scheme, host, and port visible.
type URL Substring[URLPasswordSelector]

// String returns the URL with the password replaced by the placeholder "*HIDDEN*".
func (s URL) String() string {
	return Substring[URLPasswordSelector](s).String()
}

// GoString masks the URL password when formatted with %#v.
func (s URL) GoString() string {
	return Substring[URLPasswordSelector](s).GoString()
}

// Reveal returns the raw, unmasked URL string containing the actual credentials.
func (s URL) Reveal() string {
	return Substring[URLPasswordSelector](s).Reveal()
}

// MarshalJSON serializes the URL into a JSON string containing the masked value.
func (s URL) MarshalJSON() ([]byte, error) {
	return Substring[URLPasswordSelector](s).MarshalJSON()
}
