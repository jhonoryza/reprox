package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

var regex = regexp.MustCompile(`^[a-z\d](?:[a-z\d]|-[a-z\d]){0,38}$`)
var blockList = map[string]bool{"www": true}

// stripPort removes a trailing :port from a Host header value. Browsers
// include the port for non-standard ports (e.g. example.com:4000), but
// tunnels are registered by bare hostname, so the port must be dropped
// before the tunnel lookup.
func stripPort(host string) string {
	if i := strings.LastIndex(host, ":"); i >= 0 {
		if _, err := strconv.Atoi(host[i+1:]); err == nil {
			return host[:i]
		}
	}
	return host
}

func validate(subdomain string) error {
	if len(subdomain) > 38 || len(subdomain) < 3 {
		return errors.New("subdomain length must be between 3 and 42")
	}
	if blockList[subdomain] {
		return errors.New("subdomain is in deny list")
	}
	if !regex.MatchString(subdomain) {
		return errors.New("subdomain must be lowercase & alphanumeric")
	}
	return nil
}

const letters = "abcdefghijklmnopqrstuvwxyz"

func generateRandomString(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		result[i] = letters[num.Int64()]
	}
	return string(result), nil
}

func parseHost(r io.Reader) (string, []byte, error) {
	buffer := make([]byte, 2048)
	size, err := r.Read(buffer)
	buffer = buffer[:size]
	if err != nil {
		return "", buffer, err
	}

	text := string(buffer)
	left := strings.Index(text, "Host: ")
	if left < 0 {
		left = strings.Index(text, "host: ")
	}
	if left < 0 {
		return "", buffer, fmt.Errorf("no host detected")
	}
	text = text[left+6:] // drops chars "Host: "
	right := strings.Index(text, "\n")
	if right < 0 {
		return "", buffer, fmt.Errorf("no host detected")
	}
	return strings.TrimSpace(text[:right]), buffer, nil
}

func writeResponse(conn io.WriteCloser, statusCode int, status string, message string) {
	response := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Length: %d\r\n\r\n%s", statusCode, status, len(message), message)
	conn.Write([]byte(response))
	conn.Close()
}
