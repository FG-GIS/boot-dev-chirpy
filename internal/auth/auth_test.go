package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/FG-GIS/boot-dev-chirpy/internal/auth"
	"github.com/google/uuid"
)

func TestHashing(t *testing.T) {
	pswdSlice := []string{"testWord", "testWord2", "testWord$"}
	for idx, word := range pswdSlice {
		hash, err := auth.HashPassword(word)
		if err != nil {
			t.Errorf("Error hashing the n.%d word [%s]\n", idx, word)
		}
		t.Logf("Word n.%d [%s] hashed=> %s\n", idx, word, hash)
	}
}

func TestHashCompare(t *testing.T) {
	pswdSlice := []string{"testWord", "testWord2", "testWord$"}
	hashSlice := []string{}
	for idx, word := range pswdSlice {
		hash, err := auth.HashPassword(word)
		if err != nil {
			fmt.Printf("Error hashing the n.%d word [%s]\n", idx, word)
		}
		hashSlice = append(hashSlice, hash)
	}
	for idx, test := range hashSlice {
		check, err := auth.CheckPasswordHash(pswdSlice[idx], test)
		if err != nil {
			t.Errorf("Error running Auth comparison:\n%s", err)
		}
		if check != true {
			t.Errorf("Expected true, got %v\n", check)
		} else {
			t.Logf("Test %d, succedeed.", idx)
		}
	}
}

func TestCreateJWT(t *testing.T) {
	tempo, err := time.ParseDuration("1m")
	if err != nil {
		t.Errorf("Error parsing duration: %s", err)
	}
	id := uuid.New()
	jwt, err := auth.MakeJWT(id, "secretToken", tempo)
	if err != nil {
		t.Errorf("Error creating the token: %s", err)
	}
	if jwt == "" {
		t.Logf("Error propagated, jwt is empty: %s", jwt)
	}
}

func TestValidateJWT(t *testing.T) {
	tempo1, err := time.ParseDuration("1s")
	if err != nil {
		t.Errorf("Error parsing duration: %s", err)
	}
	tempo2, err := time.ParseDuration("1m")
	if err != nil {
		t.Errorf("Error parsing duration: %s", err)
	}
	tempo3, err := time.ParseDuration("3s")
	if err != nil {
		t.Errorf("Error parsing duration: %s", err)
	}
	id := uuid.New()
	jwt, err := auth.MakeJWT(id, "secretToken", tempo1)
	if err != nil {
		t.Errorf("Error creating the token: %s", err)
	}
	time.Sleep(tempo3)
	_, err = auth.ValidateJWT(jwt, "secretToken")
	if err == nil {
		t.Error("Error, validated expired token.")
	}

	jwt, err = auth.MakeJWT(id, "secretToken", tempo2)
	if err != nil {
		t.Errorf("Error creating the token: %s", err)
	}
	_, err = auth.ValidateJWT(jwt, "secretToken")
	if err != nil {
		t.Errorf("Error validating correct token: %s", err)
	}
	jwt, err = auth.MakeJWT(id, "secretToken", tempo2)
	if err != nil {
		t.Errorf("Error creating the token: %s", err)
	}
	time.Sleep(tempo3)
	_, err = auth.ValidateJWT(jwt, "badToken")
	if err == nil {
		t.Errorf("Error validated with bad token")
	}
}
