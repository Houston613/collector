package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDurationOrInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{name: "duration in seconds", input: "15s", expected: 15, wantErr: false},
		{name: "duration in minutes", input: "2m", expected: 120, wantErr: false},
		{name: "numeric string", input: "30", expected: 30, wantErr: false},
		{name: "invalid string", input: "invalid", expected: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := parseDurationOrInt(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, val)
			}
		})
	}
}

func TestDurationOrInt_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		expected  int
		wantErr   bool
	}{
		{name: "JSON integer", jsonInput: `10`, expected: 10, wantErr: false},
		{name: "JSON duration string in seconds", jsonInput: `"15s"`, expected: 15, wantErr: false},
		{name: "JSON duration string in minutes", jsonInput: `"2m"`, expected: 120, wantErr: false},
		{name: "JSON numeric string", jsonInput: `"30"`, expected: 30, wantErr: false},
		{name: "JSON invalid string", jsonInput: `"invalid"`, expected: 0, wantErr: true},
		{name: "JSON invalid type boolean", jsonInput: `true`, expected: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d DurationOrInt
			err := json.Unmarshal([]byte(tt.jsonInput), &d)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, int(d))
			}
		})
	}
}
