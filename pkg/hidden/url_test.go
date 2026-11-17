package hidden_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/pkg/hidden"
)

func TestURL_StringAndMarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       hidden.URL
		want    string
		wantRaw string
	}{
		{
			name:    "empty url",
			s:       hidden.URL(""),
			want:    "",
			wantRaw: "",
		},
		{
			name:    "url without password",
			s:       hidden.URL("tcp://someone@example.com:1883/some-path/?query=param&another=someone%40example.com"),
			want:    "tcp://someone@example.com:1883/some-path/?query=param&another=someone%40example.com",
			wantRaw: "tcp://someone@example.com:1883/some-path/?query=param&another=someone%40example.com",
		},
		{
			name:    "url with password",
			s:       hidden.URL("tcp://someone:password@example.com:1883/some-path/?query=param&another=someone%40example.com"),
			want:    "tcp://someone:" + hidden.HiddenText + "@example.com:1883/some-path/?query=param&another=someone%40example.com",
			wantRaw: "tcp://someone:password@example.com:1883/some-path/?query=param&another=someone%40example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.s.String())
			assert.Equal(t, tt.want, fmt.Sprintf("%v", tt.s))
			assert.Equal(t, tt.want, fmt.Sprintf("%+v", tt.s))
			assert.Equal(t, `"`+tt.want+`"`, fmt.Sprintf("%#v", tt.s))
			assert.Equal(t, tt.wantRaw, tt.s.Reveal())

			data, err := tt.s.MarshalJSON()
			require.NoError(t, err)
			assert.Equal(t, `"`+tt.want+`"`, string(data))
		})
	}
}
