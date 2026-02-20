package backup

import (
	"testing"
)

func TestDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		d    Duration
		want string
	}{
		{"whole number", Duration(294), "294.0"},
		{"fractional", Duration(123.456), "123.5"},
		{"zero", Duration(0), "0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.d.MarshalJSON()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("MarshalJSON() = %s, want %s", got, tt.want)
			}
		})
	}
}
