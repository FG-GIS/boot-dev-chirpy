package auth_test

import (
	"fmt"
	"testing"

	"github.com/FG-GIS/boot-dev-chirpy/internal/auth"
)

func TestHashing(t *testing.T) {
	pswdSlice := []string{"testWord", "testWord2", "testWord$"}
	for idx, word := range pswdSlice {
		// fmt.Printf("Word %d -- %s --\n", idx, word)
		hash, err := auth.HashPassword(word)
		if err != nil {
			t.Errorf("Error hashing the n.%d word [%s]\n", idx, word)
		}
		t.Logf("Word n.%d [%s] hashed=> %s\n", idx, word, hash)
		// fmt.Printf("Word n.%d [%s] hashed=> %s\n", idx, word, hash)
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
