package credentials

import (
	"context"
	"encoding/json"

	"github.com/pchkauu/want-keep/backend/internal/connections/access"
	domain "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
)

const MaxSecretBytes = 16 * 1024 * 1024

type Repository interface {
	SaveEncryptedSecret(context.Context, domain.SecretReference, []byte) error
	EncryptedSecret(context.Context, domain.SecretReference) ([]byte, error)
}
type Vault struct {
	access *access.Service
	repo   Repository
	keys   *cryptobox.Keyring
}

func New(a *access.Service, r Repository, k *cryptobox.Keyring) *Vault {
	return &Vault{access: a, repo: r, keys: k}
}
func (v *Vault) Available() bool { return v != nil && v.keys.Available() }
func (v *Vault) Seal(ref domain.SecretReference, plain []byte) ([]byte, error) {
	if !v.Available() {
		return nil, cryptobox.ErrUnavailable
	}
	if len(plain) == 0 || len(plain) > MaxSecretBytes {
		return nil, domain.ErrSecretAccess
	}
	aad, err := v.aad(ref)
	if err != nil {
		return nil, err
	}
	return v.keys.Seal(plain, aad)
}
func (v *Vault) Open(ref domain.SecretReference, ciphertext []byte) ([]byte, error) {
	if !v.Available() {
		return nil, cryptobox.ErrUnavailable
	}
	aad, err := v.aad(ref)
	if err != nil {
		return nil, err
	}
	return v.keys.Open(ciphertext, aad)
}
func (v *Vault) aad(ref domain.SecretReference) ([]byte, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Version   int
		Purpose   string
		Reference domain.SecretReference
	}{1, "connection-secret", ref})
}
func (v *Vault) Save(ctx context.Context, token identity.Token, grantID string, purpose domain.SecretPurpose, plain []byte) error {
	if !v.Available() {
		return cryptobox.ErrUnavailable
	}
	if len(plain) < 1 || len(plain) > MaxSecretBytes {
		return domain.ErrSecretAccess
	}
	return v.access.Complete(ctx, token, grantID, purpose, func(ctx context.Context, ref domain.SecretReference) error {
		aad, err := v.aad(ref)
		if err != nil {
			return err
		}
		ciphertext, err := v.keys.Seal(plain, aad)
		if err != nil {
			return err
		}
		return v.repo.SaveEncryptedSecret(ctx, ref, ciphertext)
	})
}

// WithJobSecret lends bytes only to the concrete provider adapter for one permitted read.
// The callback runs outside database transactions and must not retain the borrowed buffer.
func (v *Vault) WithJobSecret(ctx context.Context, p household.Principal, job jobs.Job, purpose domain.SecretPurpose, use func([]byte) error) error {
	return v.WithJobReference(ctx, p, job, purpose, func(_ domain.SecretReference, plain []byte) error { return use(plain) })
}

func (v *Vault) WithJobReference(ctx context.Context, p household.Principal, job jobs.Job, purpose domain.SecretPurpose, use func(domain.SecretReference, []byte) error) error {
	if !v.Available() {
		return cryptobox.ErrUnavailable
	}
	var plain []byte
	var borrowed domain.SecretReference
	err := v.access.ForJob(ctx, p, job, purpose, func(ctx context.Context, ref domain.SecretReference) error {
		ciphertext, err := v.repo.EncryptedSecret(ctx, ref)
		if err != nil {
			return err
		}
		aad, err := v.aad(ref)
		if err != nil {
			return err
		}
		plain, err = v.keys.Open(ciphertext, aad)
		borrowed = ref
		return err
	})
	defer clear(plain)
	if err != nil {
		return err
	}
	if len(plain) == 0 || len(plain) > MaxSecretBytes {
		return domain.ErrSecretAccess
	}
	if err = v.access.ForJob(ctx, p, job, purpose, func(_ context.Context, current domain.SecretReference) error {
		if current != borrowed {
			return domain.ErrSecretAccess
		}
		return nil
	}); err != nil {
		return err
	}
	return use(borrowed, plain)
}
