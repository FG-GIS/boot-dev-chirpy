package auth_test

import (
	"net/http"
	"testing"

	"github.com/FG-GIS/boot-dev-chirpy/internal/auth"
)

func TestBearerTokenSuccess(t *testing.T) {
	headers := make(http.Header)
	headers.Add("authorization", "Bearer testtokenbearer")
	test, err := auth.GetBearerToken(headers)
	if err != nil {
		t.Errorf("Error in GetBearerToken: %s\n", err)
	}
	t.Logf("auth is: %v\n", test)
}

func TestBearerTokenFail(t *testing.T) {
	headers := make(http.Header)
	headers.Add("content-type", "text/plain")
	_, err := auth.GetBearerToken(headers)
	if err == nil {
		t.Errorf("Function did not Error as supposed")
	}
}
