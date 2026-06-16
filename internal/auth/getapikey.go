package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := strings.Split(headers.Get("Authorization"), " ")

	if len(authHeader) < 2 {
		return "", fmt.Errorf("No Authorization header found")
	}

	if authHeader[0] != "ApiKey" {
		return "", fmt.Errorf("No API Key found")
	}

	return authHeader[1], nil
}
