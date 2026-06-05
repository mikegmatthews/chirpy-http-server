package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := strings.Split(headers.Get("Authorization"), " ")

	if len(authHeader) < 2 {
		return "", fmt.Errorf("No Authorization header found")
	}

	if authHeader[0] != "Bearer" {
		return "", fmt.Errorf("No Bearer token found")
	}

	return authHeader[1], nil
}
