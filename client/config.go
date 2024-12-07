package main

import (
	"errors"
	"os"
)

type Config struct {
	Domain string
	Events string
}

func (c *Config) Load() error {
	c.Domain = os.Getenv("DOMAIN")
	if c.Domain == "" {
		return errors.New("domain is not configured")
	}
	c.Events = os.Getenv("DOMAIN_EVENT")
	if c.Events == "" {
		return errors.New("events domain is not configured")
	}
	return nil
}
