package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	DomainName        string
	MaxTunnelsPerUser int
	MaxConsPerTunnel  int
	EventServerPort   uint16
	HttpServerPort    uint16
	HttpsServerPort   uint16
	TLSCertFile       string
	TLSKeyFile        string
	EnableTLS         bool
}

func (c *Config) Load() error {
	httpPort, err := strconv.Atoi(os.Getenv("HTTP_PORT"))
	if err != nil {
		httpPort = 80
	}
	httpsPort, err := strconv.Atoi(os.Getenv("HTTPS_PORT"))
	if err != nil {
		httpsPort = 443
	}
	c.DomainName = os.Getenv("DOMAIN")
	c.MaxTunnelsPerUser = 4
	c.MaxConsPerTunnel = 24
	c.EventServerPort = 4321
	c.HttpServerPort = uint16(httpPort)
	c.HttpsServerPort = uint16(httpsPort)
	c.TLSCertFile = os.Getenv("TLS_PATH_CERT")
	c.TLSKeyFile = os.Getenv("TLS_PATH_KEY")
	c.EnableTLS = true

	if c.DomainName == "" {
		return errors.New("domain is not configured")
	}

	if c.TLSKeyFile == "" || c.TLSCertFile == "" {
		// return errors.New("TLS key/cert file is missing")
		c.EnableTLS = false
	}

	return nil
}
