package domain

import (
	"errors"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var (
	ErrConnectionNotFound = errors.New("connection not found")
	ErrInvalidConnection  = errors.New("invalid connection")
	ErrOAuthUnavailable   = errors.New("bank authorization unavailable")
	ErrOAuthAttempt       = errors.New("bank authorization attempt invalid")
	ErrOAuthUnknown       = errors.New("bank token exchange outcome unknown")
	ErrConnectionVersion  = errors.New("connection version conflict")
)

type ConnectionRecord struct {
	ID                   string
	HouseholdID          household.HouseholdID
	OwnerID              household.UserID
	Provider, State      string
	Revision, Generation uint64
	HistoryFrom          calendar.Date
	LastSuccess          time.Time
	Coverage             string
	Gaps                 []string
	Admission            Admission
	HasAdmission         bool
}

func (c ConnectionRecord) Require(p household.Principal, revision uint64, owner bool) error {
	if err := p.RequireHousehold(c.HouseholdID); err != nil {
		return err
	}
	if c.ID == "" || revision != c.Revision {
		return ErrConnectionVersion
	}
	if owner {
		return (ExternalOwnership{HouseholdID: c.HouseholdID, OwnerID: c.OwnerID}).RequireAuthentication(p)
	}
	return nil
}

type OAuthAttempt struct {
	ID, ConnectionID, SessionHash, StateHash string
	HouseholdID                              household.HouseholdID
	OwnerID                                  household.UserID
	Generation, Revision                     uint64
	ExpiresAt                                time.Time
	Phase                                    string
	Ciphertext                               []byte
}

func (a OAuthAttempt) Reference() SecretReference {
	return SecretReference{HouseholdID: a.HouseholdID, ConnectionID: a.ConnectionID, Purpose: OAuthTokens, Generation: a.Generation, Revision: a.Revision}
}

func (a OAuthAttempt) Require(p household.Principal, sessionHash string, c ConnectionRecord, now time.Time) error {
	if a.ID == "" || a.OwnerID != p.UserID() || a.HouseholdID != p.HouseholdID() || a.SessionHash != sessionHash || a.ConnectionID != c.ID || a.Generation != c.Generation || a.Revision != c.Revision || c.State == "disconnected" || !now.Before(a.ExpiresAt) {
		return ErrOAuthAttempt
	}
	return (ExternalOwnership{HouseholdID: c.HouseholdID, OwnerID: c.OwnerID}).RequireAuthentication(p)
}

type OAuthSecrets struct{ State, Nonce, Verifier string }
type TokenSet struct {
	Access    string    `json:"access_token"`
	ID        string    `json:"id_token"`
	Refresh   string    `json:"refresh_token"`
	Type      string    `json:"token_type"`
	Subject   string    `json:"subject"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RBOReport struct{ Key, AttemptID, ReportID, Phase string }
