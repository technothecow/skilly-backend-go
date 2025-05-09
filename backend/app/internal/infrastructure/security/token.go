package security

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTTL = time.Hour
	TokenCookieName = "authToken"
)

func getSecretKey() string {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		panic("JWT_SECRET_KEY is not set")
	}
	return secret
}

func CreateToken(username string, expiresAt... time.Time) (string, error) {
	secret := getSecretKey()

	exp := time.Now().Add(TokenTTL)

	if len(expiresAt) > 0 {
		exp = expiresAt[0]
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":    exp.Unix(),
	})

	return token.SignedString([]byte(secret))
}

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
	ErrInternal     = errors.New("internal error: %w")
)

func GetUsernameFromToken(tokenString string) (string, error) {
	secret := getSecretKey()

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected token signing method", ErrInternal)
		}

		return []byte(secret), nil
	})
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInternal, err.Error())
	}

	if !token.Valid {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("%w: invalid token claims", ErrInternal)
	}

	expiresAtRaw, ok := claims["exp"]
	if !ok {
		return "", fmt.Errorf("%w: token does not contain expiration time", ErrInternal)
	}

	expiresAt, ok := expiresAtRaw.(float64)
	if !ok {
		return "", fmt.Errorf("%w: invalid expiration time", ErrInternal)
	}

	if time.Now().Unix() > int64(expiresAt) {
		return "", ErrExpiredToken
	}

	username, ok := claims["username"]
	if !ok {
		return "", fmt.Errorf("%w: token does not contain username", ErrInternal)
	}

	usernameStr, ok := username.(string)
	if !ok {
		return "", fmt.Errorf("%w: invalid username", ErrInternal)
	}

	return usernameStr, nil
}
