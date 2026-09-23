package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct{ Secret []byte }

func (j JWT) Sign(subject string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.Secret)
}

func (j JWT) Verify(token string) (string, error) {
	claims := new(jwt.RegisteredClaims)
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return j.Secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || claims.Subject == "" {
		return "", errors.New("invalid token")
	}
	return claims.Subject, nil
}
