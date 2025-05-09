package security

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTTL = time.Hour
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

func GetUsernameFromToken(tokenString string) (string, error) {
	secret := getSecretKey()

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected token signing method")
		}

		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	expiresAtRaw, ok := claims["exp"]
	if !ok {
		return "", errors.New("token does not contain expiration time")
	}

	expiresAt, ok := expiresAtRaw.(float64)
	if !ok {
		return "", errors.New("invalid expiration time")
	}

	if time.Now().Unix() > int64(expiresAt) {
		return "", errors.New("token expired")
	}

	username, ok := claims["username"]
	if !ok {
		return "", errors.New("token does not contain username")
	}

	usernameStr, ok := username.(string)
	if !ok {
		return "", errors.New("invalid username")
	}

	return usernameStr, nil
}
