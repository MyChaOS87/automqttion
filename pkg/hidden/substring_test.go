package hidden_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/pkg/hidden"
)

type SubstringInterface interface {
	String() string
	GoString() string
	Reveal() string
	MarshalJSON() ([]byte, error)
}

type aaMatcher struct{}

func (aaMatcher) Selectors() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile("aa"),
	}
}

type axaMatcher struct{}

func (axaMatcher) Selectors() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile("a(x+)a"),
	}
}

type greedyAXAMatcher struct{}

func (greedyAXAMatcher) Selectors() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile("a(x*)a"),
	}
}

type nilMatcher struct{}

func (nilMatcher) Selectors() []*regexp.Regexp {
	return nil
}

type innerNilMatcher struct{}

func (innerNilMatcher) Selectors() []*regexp.Regexp {
	return []*regexp.Regexp{
		nil,
	}
}

type nestedMatcher struct{}

func (nestedMatcher) Selectors() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile("(a(b))"),
	}
}

func TestSubstring_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       SubstringInterface
		want    string
		wantRaw string
	}{
		{
			name: "empty substring",
			s:    hidden.Substring[aaMatcher](""),
			want: "",
		},
		{
			name:    "substring without match",
			s:       hidden.Substring[aaMatcher]("hello world"),
			want:    "hello world",
			wantRaw: "hello world",
		},
		{
			name:    "substring with match",
			s:       hidden.Substring[aaMatcher]("hello aa world"),
			want:    "hello " + hidden.HiddenText + " world",
			wantRaw: "hello aa world",
		},
		{
			name:    "substring with multiple matches",
			s:       hidden.Substring[aaMatcher]("aa hello aa world aa"),
			want:    hidden.HiddenText + " hello " + hidden.HiddenText + " world " + hidden.HiddenText,
			wantRaw: "aa hello aa world aa",
		},
		{
			name:    "substring with overlapping matches",
			s:       hidden.Substring[aaMatcher]("aaaa hello aaa world"),
			want:    hidden.HiddenText + hidden.HiddenText + " hello " + hidden.HiddenText + "a world",
			wantRaw: "aaaa hello aaa world",
		},
		{
			name:    "substring with complex match",
			s:       hidden.Substring[axaMatcher]("aaxxxxaa hello axaa world"),
			want:    "aa" + hidden.HiddenText + "aa hello a" + hidden.HiddenText + "aa world",
			wantRaw: "aaxxxxaa hello axaa world",
		},
		{
			name:    "substring with strange match",
			s:       hidden.Substring[greedyAXAMatcher]("aaxaa hello axaa world"),
			want:    "aaxaa hello a" + hidden.HiddenText + "aa world",
			wantRaw: "aaxaa hello axaa world",
		},
		{
			name:    "substring with nil matcher",
			s:       hidden.Substring[nilMatcher]("hello world"),
			want:    "hello world",
			wantRaw: "hello world",
		},
		{
			name:    "substring with inner nil matcher",
			s:       hidden.Substring[innerNilMatcher]("hello world"),
			want:    "hello world",
			wantRaw: "hello world",
		},
		{
			name:    "substring with nested capture groups",
			s:       hidden.Substring[nestedMatcher]("ab"),
			want:    hidden.HiddenText,
			wantRaw: "ab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.s.String())
			assert.Equal(t, `"`+tt.want+`"`, fmt.Sprintf("%#v", tt.s))
			assert.Equal(t, tt.wantRaw, tt.s.Reveal())

			data, err := tt.s.MarshalJSON()
			require.NoError(t, err)
			assert.Equal(t, `"`+tt.want+`"`, string(data))
		})
	}
}
