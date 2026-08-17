package netutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSubnet(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"CIDR v4", "192.168.1.0/24", false},
		{"Single IPv4", "192.168.1.100", false},
		{"Single IPv6", "::1", false},
		{"CIDR v6", "2001:db8::/32", false},
		{"Invalid", "not-an-ip", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ParseSubnet(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, res)
			}
		})
	}
}
