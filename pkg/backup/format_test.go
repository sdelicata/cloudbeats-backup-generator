package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuration_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    Duration
		want string
	}{
		{"whole number", Duration(294), "294.0"},
		{"fractional", Duration(123.456), "123.5"},
		{"zero", Duration(0), "0.0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.d.MarshalJSON()
			require.NoError(t, err)
			assert.Equal(t, test.want, string(got))
		})
	}
}
