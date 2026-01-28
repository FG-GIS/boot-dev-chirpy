package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	hashedP, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hashedP, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	flag, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return flag, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	currentTime := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(currentTime),
		ExpiresAt: jwt.NewNumericDate(currentTime.Add(expiresIn)),
		Subject:   userID.String(),
	})
	strTkn, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return strTkn, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	tkn, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	subject, err := tkn.Claims.GetSubject()
	if err != nil {
		return uuid.UUID{}, err
	}
	out, err := uuid.Parse(subject)
	if err != nil {
		return uuid.UUID{}, err
	}
	return out, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	auth, ok := headers["Authorization"]
	if !ok {
		return "", fmt.Errorf("Error, authorization header missing.")
	}
	return strings.Replace(string(auth[0]), "Bearer ", "", 1), nil
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}
