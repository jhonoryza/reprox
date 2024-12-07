package config

import (
	"errors"
	"os"
)

type Config struct {
	DomainName        string
	MaxTunnelsPerUser int
	MaxConsPerTunnel  int
	EventServerPort   uint16
	HttpServerPort    uint16
}

func (c *Config) Load() error {
	c.DomainName = os.Getenv("DOMAIN")
	c.MaxTunnelsPerUser = 4
	c.MaxConsPerTunnel = 24
	c.EventServerPort = 4321
	c.HttpServerPort = 80

	if c.DomainName == "" {
		return errors.New("domain is not configured")
	}

	return nil
}
