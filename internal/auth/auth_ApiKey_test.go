package auth_test

import (
	"net/http"
	"testing"

	"github.com/FG-GIS/boot-dev-chirpy/internal/auth"
)

func TestSuccess(t *testing.T) {
	headers := make(http.Header)
	apiKey := "testThisApiKey"
	headers.Add("authorization", "ApiKey "+apiKey)
	test, err := auth.GetAPIKey(headers)
	if err != nil || test != apiKey {
		t.Errorf("Error in GetAPIKey: %s\n", err)
	}
	t.Logf("auth is: %v\n", test)
}

func TestFailNoHeader(t *testing.T) {
	headers := make(http.Header)
	headers.Add("content-type", "text/plain")
	_, err := auth.GetBearerToken(headers)
	if err == nil {
		t.Errorf("Function did not Error as supposed")
	}
}

func TestFailBadKey(t *testing.T) {
	headers := make(http.Header)
	apiKey := "testThisApiKey"
	headers.Add("authorization", "ApiKey ")
	test, err := auth.GetAPIKey(headers)
	if test == apiKey {
		t.Errorf("Error in GetAPIKey: %s\n", err)
	}
	t.Logf("auth is: %v\n", test)
}
