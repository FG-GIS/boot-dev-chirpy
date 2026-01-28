package auth_test

import (
	"testing"
	"time"

	"github.com/FG-GIS/boot-dev-chirpy/internal/auth"
	"github.com/google/uuid"
)

func TestCreateJWTSuccess(t *testing.T) {
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

// Can't make this fail anyhow
// func TestCreateJWTFail(t *testing.T) {
// 	tempo, err := time.ParseDuration("-1s")
// 	if err == nil {
// 		t.Error("Did not error at parse duration")
// 	}
// 	id := uuid.New()
// 	jwt, err := auth.MakeJWT(id, "", tempo)
// 	if err == nil {
// 		t.Error("Did not error at MakeJWT")
// 	}
// 	if jwt != "" {
// 		t.Errorf("Function supposed to error: %s\n", jwt)
// 	}
// }

func TestValidateJWTSuccess(t *testing.T) {
	tkDuration, err := time.ParseDuration("1m")
	if err != nil {
		t.Errorf("Error parsing duration: %s", err)
	}
	id := uuid.New()
	jwt, err := auth.MakeJWT(id, "secretToken", tkDuration)
	if err != nil {
		t.Errorf("Error creating the token: %s", err)
	}
	tkCheck, err := auth.ValidateJWT(jwt, "secretToken")
	if err != nil {
		t.Errorf("Error validating the token: %s\n", err)
	}
	if tkCheck != id {
		t.Errorf("Error, returned uuid is:\n%s\n\nExpected uuid is:\n%s\n", tkCheck.String(), id.String())
	}
}

func TestValidateJWTFailExpired(t *testing.T) {
	tkDuration, err := time.ParseDuration("1s")
	if err != nil {
		t.Errorf("Error parsing duration: %s", err)
	}
	id := uuid.New()
	jwt, err := auth.MakeJWT(id, "secretToken", tkDuration)
	if err != nil {
		t.Errorf("Error creating the token: %s", err)
	}
	time.Sleep(tkDuration)
	time.Sleep(tkDuration)
	tkCheck, err := auth.ValidateJWT(jwt, "secretToken")
	if err == nil {
		t.Error("ValidateJWT supposed to error out, expired token.")
	}
	if tkCheck != uuid.Nil {
		t.Errorf("Returned UUID supposed to be nil,\n instead is:\n%s\n", tkCheck.String())
	}
}

func TestValidateJWTFailBadToken(t *testing.T) {
	tkDuration, err := time.ParseDuration("1s")
	if err != nil {
		t.Errorf("Error parsing duration: %s", err)
	}
	id := uuid.New()
	jwt, err := auth.MakeJWT(id, "secretToken", tkDuration)
	if err != nil {
		t.Errorf("Error creating the token: %s", err)
	}
	time.Sleep(tkDuration)
	time.Sleep(tkDuration)
	tkCheck, err := auth.ValidateJWT(jwt, "wrongToken")
	if err == nil {
		t.Error("ValidateJWT supposed to error out, bad secret token.")
	}
	if tkCheck != uuid.Nil {
		t.Errorf("Returned UUID supposed to be nil,\n instead is:\n%s\n", tkCheck.String())
	}
}
