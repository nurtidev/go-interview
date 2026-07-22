// Package auth provides password hashing (bcrypt), JWT issuing/parsing
// (HS256, 30-day expiry) and a net/http authentication middleware.
package auth

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// TokenTTL is the lifetime of an issued JWT.
const TokenTTL = 30 * 24 * time.Hour

// Service issues and validates tokens against a shared secret.
type Service struct {
	secret []byte
}

// New builds a Service from the HS256 signing secret.
func New(secret string) *Service {
	return &Service{secret: []byte(secret)}
}

// HashPassword returns the bcrypt hash of pw.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword reports whether pw matches the stored bcrypt hash.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// Claims are the JWT claims carried in a token.
type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken issues a signed JWT for the given user.
func (s *Service) GenerateToken(userID int64, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// parse validates a raw token string and returns its claims.
func (s *Service) parse(raw string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

type ctxKey int

const userIDKey ctxKey = iota

// Middleware rejects requests without a valid Bearer token and injects the
// authenticated user id into the request context.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeUnauthorized(w)
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims, err := s.parse(raw)
		if err != nil {
			writeUnauthorized(w)
			return
		}
		uid, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			writeUnauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserID extracts the authenticated user id from a request context.
func UserID(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(userIDKey).(int64)
	return v, ok
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}
