package main

import (
	"errors"
	"log"
	"os"
	"strconv"
)

type Config struct {
	DomainName      string
	EventServerPort uint16
	HttpServerPort  uint16
	HttpsServerPort uint16
	TLSCertFile     string
	TLSKeyFile      string
	EnableTLS       bool
}

func (c *Config) Load() error {
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "80"
	}
	httpsPort := os.Getenv("HTTPS_PORT")
	if httpsPort == "" {
		httpsPort = "443"
	}
	httpPortInt, err := strconv.Atoi(httpPort)
	if err != nil {
		return err
	}
	httpsPortInt, err := strconv.Atoi(httpsPort)
	if err != nil {
		return err
	}

	c.DomainName = os.Getenv("DOMAIN")
	c.EventServerPort = 4321
	c.HttpServerPort = uint16(httpPortInt)
	c.HttpsServerPort = uint16(httpsPortInt)
	c.TLSCertFile = os.Getenv("TLS_PATH_CERT")
	c.TLSKeyFile = os.Getenv("TLS_PATH_KEY")
	c.EnableTLS = true

	if c.DomainName == "" {
		return errors.New("domain is not configured")
	}

	if c.TLSKeyFile == "" || c.TLSCertFile == "" {
		c.EnableTLS = false
	}

	log.Printf("http:%d https:%d event:%d \n", httpPortInt, httpsPortInt, c.EventServerPort)
	log.Printf("TLS enabled: %v", c.EnableTLS)

	return nil
}
