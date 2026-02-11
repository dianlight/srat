package urlutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name         string
		urlStr       string
		allowedHosts []string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid github url",
			urlStr:       "https://github.com/dianlight/srat/releases/download/v1.0.0/srat.zip",
			allowedHosts: []string{"github.com", "objects.githubusercontent.com"},
			wantErr:      false,
		},
		{
			name:         "valid objects github user content url",
			urlStr:       "https://objects.githubusercontent.com/github-production-release-asset-2e65be/12345/srat.zip",
			allowedHosts: []string{"github.com", "objects.githubusercontent.com"},
			wantErr:      false,
		},
		{
			name:         "insecure http scheme",
			urlStr:       "http://github.com/dianlight/srat/releases/download/v1.0.0/srat.zip",
			allowedHosts: []string{"github.com"},
			wantErr:      true,
			errContains:  "insecure URL scheme",
		},
		{
			name:         "untrusted host",
			urlStr:       "https://malicious.com/srat.zip",
			allowedHosts: []string{"github.com"},
			wantErr:      true,
			errContains:  "untrusted host: malicious.com",
		},
		{
			name:         "invalid url format",
			urlStr:       "https://github.com/%%invalid",
			allowedHosts: []string{"github.com"},
			wantErr:      true,
			errContains:  "invalid URL",
		},
		{
			name:         "empty allowed hosts",
			urlStr:       "https://github.com/srat.zip",
			allowedHosts: []string{},
			wantErr:      true,
			errContains:  "untrusted host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.urlStr, tt.allowedHosts)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
