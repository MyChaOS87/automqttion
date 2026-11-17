package hidden_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/pkg/hidden"
)

func TestString_StringAndMarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		s        hidden.String
		want     string
		wantRaw  string
		wantJSON string
	}{
		{
			name:     "empty string",
			s:        hidden.String(""),
			want:     "",
			wantRaw:  "",
			wantJSON: `""`,
		},
		{
			name:     "non-empty string",
			s:        hidden.String("some sensitive data"),
			want:     hidden.HiddenText,
			wantRaw:  "some sensitive data",
			wantJSON: `"` + hidden.HiddenText + `"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.s.String())
			assert.Equal(t, tt.want, fmt.Sprintf("%v", tt.s))
			assert.Equal(t, tt.want, fmt.Sprintf("%+v", tt.s))
			assert.Equal(t, tt.wantJSON, fmt.Sprintf("%#v", tt.s))
			assert.Equal(t, tt.wantRaw, tt.s.Reveal())

			data, err := tt.s.MarshalJSON()
			require.NoError(t, err)
			assert.Equal(t, tt.wantJSON, string(data))
		})
	}
}
