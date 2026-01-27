package auth_test

import (
	"testing"

	"github.com/FG-GIS/boot-dev-chirpy/internal/auth"
)

func TestHashingSuccess(t *testing.T) {
	pswdSlice := []string{"testWord", "testWord2", "testWord$"}
	for idx, word := range pswdSlice {
		hash, err := auth.HashPassword(word)
		if err != nil {
			t.Errorf("Error hashing the n.%d word [%s]\n", idx, word)
		}
		t.Logf("Word n.%d [%s] hashed=> %s\n", idx, word, hash)
	}
}

// Can't make this fail
// func TestHashingFail(t *testing.T) {
// 	pswd := ""
// 	_, err := auth.HashPassword(pswd)
// 	if err == nil {
// 		t.Errorf("Error HashPassword did not error out")
// 	}
// }

func TestHashCompareSuccess(t *testing.T) {
	pswdSlice := []string{"testWord", "testWord2", "testWord$"}
	hashSlice := []string{}
	for idx, word := range pswdSlice {
		hash, err := auth.HashPassword(word)
		if err != nil {
			t.Errorf("Error hashing the n.%d word [%s]\n", idx, word)
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
		}
	}
}

func TestHashCompareFail(t *testing.T) {
	pswdSlice := []string{"testWord", "testWord2", "testWord$"}
	wrongSlice := []string{"badWord", "badWord2", "badWord$"}
	hashSlice := []string{}
	for idx, word := range pswdSlice {
		hash, err := auth.HashPassword(word)
		if err != nil {
			t.Errorf("Error hashing the n.%d word [%s]\n", idx, word)
		}
		hashSlice = append(hashSlice, hash)
	}
	for idx, test := range hashSlice {
		check, err := auth.CheckPasswordHash(wrongSlice[idx], test)
		if err != nil {
			t.Errorf("Error running Auth comparison:\n%s", err)
		}
		if check == true {
			t.Errorf("Expected 'false', got %v\n", check)
		}
	}
}
