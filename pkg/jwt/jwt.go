package jwt

import (
	"context"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// AuthenticationType indicates a JWT.
const AuthenticationType = "JWT"

type JWT interface {
	// Sign signs jwt token.
	Sign(ctx context.Context, payload map[string]interface{}) (tokenString string, expiry time.Time, err error)

	// Parse parses token string to jwt.
	Parse(ctx context.Context, tokenString string, ignoreExpiration bool) (token *jwt.Token, expiry time.Time, err error)
}

type internalHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
