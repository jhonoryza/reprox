package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"regexp"
)

var regex = regexp.MustCompile(`^[a-z\d](?:[a-z\d]|-[a-z\d]){0,38}$`)
var blockList = map[string]bool{"www": true}

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

func generateRandomString(length int) (string, error) {
	// Mengalokasikan buffer byte
	randomBytes := make([]byte, length)

	// Mengisi buffer dengan data acak
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Mengubah byte menjadi string (base64 untuk representasi aman)
	return base64.RawURLEncoding.EncodeToString(randomBytes)[:length], nil
}
