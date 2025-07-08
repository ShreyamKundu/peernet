package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken generates a new JWT for a given peer ID.
func GenerateToken(peerID string, secretKey string) (string, error) {
	claims := jwt.MapClaims{
		"sub": peerID,                               // Subject (Peer ID)
		"iat": time.Now().Unix(),                    // Issued At
 		"exp": time.Now().Add(time.Hour * 24).Unix(), // Expiration Time
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}


// ValidateToken validates a JWT string and returns the peer ID (subject).
func ValidateToken(tokenString string, secretKey string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		peerID := claims["sub"].(string)
		return peerID, nil
	}

	return "", fmt.Errorf("invalid token")
}