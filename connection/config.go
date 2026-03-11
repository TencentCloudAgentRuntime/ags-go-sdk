package connection

import (
	"net/http"
	"net/url"
)

const (
	//nolint:unused // reserved for future use
	userAgent = "agentsandbox-go-sdk"

	// DefaultScheme is the default URL scheme for data plane connections.
	DefaultScheme = "https"
)

type Config struct {
	Domain      string
	AccessToken string
	Headers     http.Header
	Proxy       *url.URL
	// Scheme specifies the URL scheme for data plane connections ("http" or "https").
	// Default is "https" when empty.
	Scheme string
}

// GetScheme returns the configured scheme, defaulting to "https" if not set.
func (c *Config) GetScheme() string {
	if c.Scheme == "" {
		return DefaultScheme
	}
	return c.Scheme
}

func NewConfig() *Config {
	return &Config{}
}
