package jwt

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/code-and-chill/auth-api/pkg/timegenerator"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type RS256 struct {
	timegen         timegenerator.TimeGenerator
	keyID           string
	issuer          string
	audience        string
	maxAge          time.Duration
	privateKey      *rsa.PrivateKey
	publicKey       *rsa.PublicKey
	publicKeyURL    *string
	httpClient      internalHTTPClient
	cachedPublicKey sync.Map
}

func (R *RS256) Sign(_ context.Context, payload map[string]interface{}) (tokenString string, expiry time.Time, err error) {
	if R.privateKey == nil {
		return "", time.Time{}, errors.New("no private key provided")
	}
	now := R.timegen.Now().UTC()
	expiresAt := now.Add(R.maxAge).UTC()
	payload["iss"] = R.issuer
	payload["aud"] = R.audience
	payload["auth_time"] = now.Unix()
	payload["iat"] = now.Unix()
	payload["exp"] = expiresAt.Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(payload))
	token.Header["kid"] = R.keyID
	tokenString, err = token.SignedString(R.privateKey)
	if err != nil {
		return "", time.Time{}, errors.WithStack(err)
	}
	return tokenString, expiresAt, nil
}

// Parse parses token string to jwt.
func (R *RS256) Parse(ctx context.Context, tokenString string, ignoreExpiration bool) (token *jwt.Token, expiry time.Time, err error) {
	token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if err := R.validateHeaders(token); err != nil {
			return nil, errors.WithStack(err)
		}
		if err := R.validateClaims(token.Claims, ignoreExpiration); err != nil {
			return nil, errors.WithStack(err)
		}
		kid := token.Header["kid"].(string)
		publicKey, err := R.getPublicKey(ctx, kid)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return publicKey, nil
	})
	if err != nil {
		if err.Error() == "Token is expired" && !ignoreExpiration {
			err = errors.WithStack(err)
			return
		} else if err.Error() != "Token is expired" {
			err = errors.WithStack(err)
			return
		}
	}
	claims := token.Claims.(jwt.MapClaims)
	if claims == nil || claims["exp"] == nil {
		return nil, time.Time{}, errors.New("invalid claims: invalid expiration")
	}
	switch expirationTime := claims["exp"].(type) {
	case int64:
		expiry = time.Unix(expirationTime, 0).UTC()
	case int32:
		expiry = time.Unix(int64(expirationTime), 0).UTC()
	case int:
		expiry = time.Unix(int64(expirationTime), 0).UTC()
	case float64:
		expiry = time.Unix(int64(expirationTime), 0).UTC()
	case float32:
		expiry = time.Unix(int64(expirationTime), 0).UTC()
	}
	return token, expiry, nil
}

func (R *RS256) getPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	if R.publicKey != nil {
		return R.publicKey, nil
	}
	if R.publicKeyURL == nil {
		return nil, errors.New("no public key URL provided")
	}

	cacheKey := fmt.Sprintf("%s:%s", *R.publicKeyURL, kid)
	if rawPublicKey, ok := R.cachedPublicKey.Load(cacheKey); ok {
		switch publicKeyByte := rawPublicKey.(type) {
		case []byte:
			return jwt.ParseRSAPublicKeyFromPEM(publicKeyByte)
		}
	}
	R.cachedPublicKey.Delete(cacheKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *R.publicKeyURL, nil)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	resp, err := R.httpClient.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err.Error())
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		result := make(map[string]string)
		err = json.Unmarshal(bodyBytes, &result)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if resultKID, ok := result[kid]; ok {
			return jwt.ParseRSAPublicKeyFromPEM([]byte(resultKID))
		}
	} else {
		err = errors.Errorf("failed getting public key: http status %d", resp.StatusCode)
	}
	err = errors.Errorf("failed getting public key: kid %s is not found", kid)
	return nil, errors.WithStack(err)
}

func (R *RS256) validateHeaders(token *jwt.Token) error {
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		err := errors.Errorf("invalid signing method [%v]", token.Header["alg"])
		return errors.WithStack(err)
	}
	var hasValidType bool
	switch typ := token.Header["type"].(type) {
	case string:
		hasValidType = typ == AuthenticationType
	}
	if !hasValidType {
		err := errors.Errorf("invalid signing type [%v]", token.Header["typ"])
		return errors.WithStack(err)
	}
	return nil
}

func (R *RS256) validateClaims(jwtClaims jwt.Claims, ignoreExpiration bool) error {
	if !ignoreExpiration {
		if err := jwtClaims.Valid(); err != nil {
			return errors.WithStack(err)
		}
	}
	claims := jwtClaims.(jwt.MapClaims)
	var hasValidIssuer bool
	switch issuer := claims["iss"].(type) {
	case string:
		hasValidIssuer = issuer == R.issuer
	}
	if !hasValidIssuer {
		err := errors.Errorf("invalid issuer [%v]", claims["iss"])
		return errors.WithStack(err)
	}
	var hasValidAudience bool
	switch audience := claims["aud"].(type) {
	case string:
		hasValidAudience = audience == R.audience
	}
	if !hasValidAudience {
		err := errors.Errorf("invalid audience [%v]", claims["aud"])
		return errors.WithStack(err)
	}
	return nil
}

// NewRS256 instantiate a new RS256.
func NewRS256(timegen timegenerator.TimeGenerator, keyID, issuer, audience string,
	privateKey, publicKey *[]byte, publicKeyURL *string, maxAge time.Duration, httpClient internalHTTPClient) (JWT, error) {

	var signKey *rsa.PrivateKey
	var verifyKey *rsa.PublicKey
	var err error
	if privateKey != nil {
		signKey, err = jwt.ParseRSAPrivateKeyFromPEM(*privateKey)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	if publicKey != nil {
		verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(*publicKey)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return &RS256{
		timegen:         timegen,
		keyID:           keyID,
		issuer:          issuer,
		audience:        audience,
		maxAge:          maxAge,
		privateKey:      signKey,
		publicKey:       verifyKey,
		publicKeyURL:    publicKeyURL,
		httpClient:      httpClient,
		cachedPublicKey: sync.Map{},
	}, nil
}
