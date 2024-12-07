package config

type Config struct {
	DomainName        string
	MaxTunnelsPerUser int
	MaxConsPerTunnel  int
	EventServerPort   uint16
}

func (c *Config) Load() error {
	c.DomainName = "me.localhost"
	c.MaxTunnelsPerUser = 4
	c.MaxConsPerTunnel = 24
	c.EventServerPort = 4321

	return nil
}
