package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		})
	if err != nil {
		return uuid.Nil, err
	} else if subject, err := token.Claims.GetSubject(); err == nil {
		return uuid.Parse(subject)
	} else {
		return uuid.Nil, fmt.Errorf("Error parsing JWT claims: %s\n", err)
	}
}
