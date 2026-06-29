package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	IsAdmin     bool     `json:"is_admin"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

type TokenService struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenService(secret string, expiryHours int) (*TokenService, error) {
	if secret == "" {
		secret = "dev-insecure-change-me-in-production"
	}
	if expiryHours <= 0 {
		expiryHours = 24
	}
	return &TokenService{
		secret: []byte(secret),
		ttl:    time.Duration(expiryHours) * time.Hour,
	}, nil
}

func (t *TokenService) Sign(userID uuid.UUID, email, name string, isAdmin bool, permissions []string) (string, error) {
	claims := Claims{
		UserID:      userID.String(),
		Email:       email,
		Name:        name,
		IsAdmin:     isAdmin,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.secret)
}

func (t *TokenService) Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return t.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
