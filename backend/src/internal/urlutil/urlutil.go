package urlutil

import (
	"fmt"
	"net/url"
	"slices"
)

// ValidateURL checks if a URL is safe to request by verifying it uses HTTPS
// and its host matches one of the allowed hosts.
func ValidateURL(urlStr string, allowedHosts []string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "https" {
		return fmt.Errorf("insecure URL scheme: %s (must be https)", u.Scheme)
	}

	host := u.Hostname()
	if slices.Contains(allowedHosts, host) {
		return nil
	}

	return fmt.Errorf("untrusted host: %s", host)
}
