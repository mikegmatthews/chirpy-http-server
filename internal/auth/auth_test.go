package auth

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}

	secret := rand.Text()
	jwt, err := MakeJWT(uuid, secret, time.Second*1)
	if err != nil {
		t.Fatal(err)
	}

	validUUID, err := ValidateJWT(jwt, secret)
	if err != nil {
		t.Fatal(err)
	}

	if validUUID != uuid {
		t.Fatalf("JWT not valid; UUIDs don't match. Expected: %s, Got: %s\n",
			uuid.String(), validUUID.String())
	}
}

func TestInvalidJWT(t *testing.T) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}

	secret := rand.Text()
	wrongSecret := rand.Text()
	jwt, err := MakeJWT(uuid, wrongSecret, time.Second*1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ValidateJWT(jwt, secret)
	if err == nil {
		t.Fatal("Wrong secret did not create an invalid JWT.")
	}
}

func TestExpiredJWT(t *testing.T) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}

	secret := rand.Text()
	jwt, err := MakeJWT(uuid, secret, time.Second*0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ValidateJWT(jwt, secret)
	if err == nil {
		t.Fatal("Expired JWT was not invalidated.")
	}
}
