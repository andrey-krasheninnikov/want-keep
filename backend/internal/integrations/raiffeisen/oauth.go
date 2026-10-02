package raiffeisen

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
)

// OAuthConfig is operator-owned. Legacy RBO has no established discovery endpoint.
// A confirmed issuer and trusted public keys are required before Code Flow can run.
type OAuthConfig struct {
	ClientID, ClientSecret, RedirectURI, Issuer string
	Keys                                        map[string]*rsa.PublicKey
}
type Authorizer struct {
	Config OAuthConfig
	Client *Client
	Now    func() time.Time
}

func (a *Authorizer) validationTime() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func LoadOAuth(clientFile, secretFile, issuer, jwksFile, redirect string) (*Authorizer, error) {
	id, err := os.ReadFile(clientFile)
	if err != nil {
		return nil, connections.ErrOAuthUnavailable
	}
	defer clear(id)
	info, err := os.Stat(secretFile)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, connections.ErrOAuthUnavailable
	}
	secret, err := os.ReadFile(secretFile)
	if err != nil {
		return nil, connections.ErrOAuthUnavailable
	}
	defer clear(secret)
	keysData, err := os.ReadFile(jwksFile)
	if err != nil {
		return nil, connections.ErrOAuthUnavailable
	}
	var keys struct {
		Keys []struct {
			ID        string `json:"kid"`
			Type      string `json:"kty"`
			Use       string `json:"use"`
			Algorithm string `json:"alg"`
			N         string `json:"n"`
			E         string `json:"e"`
		} `json:"keys"`
	}
	if json.Unmarshal(keysData, &keys) != nil || len(keys.Keys) == 0 || len(keys.Keys) > 32 {
		return nil, connections.ErrOAuthUnavailable
	}
	config := OAuthConfig{ClientID: strings.TrimSpace(string(id)), ClientSecret: strings.TrimSpace(string(secret)), Issuer: issuer, RedirectURI: redirect, Keys: map[string]*rsa.PublicKey{}}
	for _, key := range keys.Keys {
		n, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			return nil, connections.ErrOAuthUnavailable
		}
		e, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil || len(e) > 4 || len(e) == 0 || key.ID == "" || key.Type != "RSA" || key.Use != "sig" || key.Algorithm != "RS256" || config.Keys[key.ID] != nil {
			return nil, connections.ErrOAuthUnavailable
		}
		exponent := new(big.Int).SetBytes(e)
		k := &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent.Int64())}
		if k.N.BitLen() < 2048 || k.E < 3 || k.E%2 == 0 {
			return nil, connections.ErrOAuthUnavailable
		}
		config.Keys[key.ID] = k
	}
	a := &Authorizer{Config: config, Client: NewClient(nil)}
	if !a.Available() {
		return nil, connections.ErrOAuthUnavailable
	}
	return a, nil
}
func (a *Authorizer) Available() bool {
	if a == nil || a.Client == nil || a.Config.ClientID == "" || a.Config.ClientSecret == "" || len(a.Config.Keys) == 0 {
		return false
	}
	u, err := url.Parse(a.Config.Issuer)
	if err != nil || u.Scheme != "https" || u.Host != "sso.rbo.raiffeisen.ru" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	return a.Config.RedirectURI == "https://want-keep.tech/api/v1/connections/raiffeisen/callback"
}
func (a *Authorizer) URL(secrets connections.OAuthSecrets) (string, error) {
	if !a.Available() || len(secrets.State) < 32 || len(secrets.Nonce) < 32 || len(secrets.Verifier) < 43 {
		return "", connections.ErrOAuthUnavailable
	}
	challenge := sha256.Sum256([]byte(secrets.Verifier))
	values := url.Values{"client_id": {a.Config.ClientID}, "redirect_uri": {a.Config.RedirectURI}, "response_type": {"code"}, "scope": {"openid profile email phone"}, "state": {secrets.State}, "nonce": {secrets.Nonce}, "prompt": {"login"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}, "code_challenge_method": {"S256"}}
	return authorizeURL + "?" + values.Encode(), nil
}

type idClaims struct {
	jwt.RegisteredClaims
	Nonce           string `json:"nonce"`
	AccessHash      string `json:"at_hash"`
	AuthorizedParty string `json:"azp"`
}

func (a *Authorizer) validate(t Tokens, nonce, subject string, now time.Time) (connections.TokenSet, error) {
	claims := &idClaims{}
	parsed, err := jwt.ParseWithClaims(t.ID, claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || a.Config.Keys[kid] == nil {
			return nil, ErrResponse
		}
		return a.Config.Keys[kid], nil
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(a.Config.Issuer), jwt.WithAudience(a.Config.ClientID), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil || !parsed.Valid || claims.Subject == "" || claims.IssuedAt == nil || claims.IssuedAt.Time.After(now) || claims.ExpiresAt.Time.After(now.Add(24*time.Hour)) || nonce != "" && claims.Nonce != nonce || subject != "" && claims.Subject != subject || len(claims.Audience) > 1 && claims.AuthorizedParty != a.Config.ClientID || claims.AuthorizedParty != "" && claims.AuthorizedParty != a.Config.ClientID {
		return connections.TokenSet{}, ErrResponse
	}
	if claims.AccessHash != "" {
		hash := sha256.Sum256([]byte(t.Access))
		if claims.AccessHash != base64.RawURLEncoding.EncodeToString(hash[:16]) {
			return connections.TokenSet{}, ErrResponse
		}
	}
	return connections.TokenSet{Access: t.Access, ID: t.ID, Refresh: t.Refresh, Type: t.Type, Subject: claims.Subject, IssuedAt: now, ExpiresAt: claims.ExpiresAt.Time}, nil
}
func (a *Authorizer) Exchange(ctx context.Context, code string, s connections.OAuthSecrets) (connections.TokenSet, error) {
	if !a.Available() {
		return connections.TokenSet{}, connections.ErrOAuthUnavailable
	}
	t, err := a.Client.Exchange(ctx, a.Config.ClientID, a.Config.ClientSecret, code, s.Verifier, a.Config.RedirectURI)
	if err != nil {
		return connections.TokenSet{}, oauthError(err)
	}
	result, err := a.validate(t, s.Nonce, "", a.validationTime())
	if err != nil {
		return result, connections.ErrOAuthUnknown
	}
	return result, nil
}
func (a *Authorizer) Refresh(ctx context.Context, old connections.TokenSet) (connections.TokenSet, error) {
	if !a.Available() {
		return connections.TokenSet{}, connections.ErrOAuthUnavailable
	}
	t, err := a.Client.Refresh(ctx, a.Config.ClientID, a.Config.ClientSecret, old.Refresh)
	if err != nil {
		return connections.TokenSet{}, oauthError(err)
	}
	result, err := a.validate(t, "", old.Subject, a.validationTime())
	if err != nil {
		return result, connections.ErrOAuthUnknown
	}
	return result, nil
}
func oauthError(err error) error {
	if err == ErrUnknown {
		return connections.ErrOAuthUnknown
	}
	return connections.ErrSecretAccess
}
