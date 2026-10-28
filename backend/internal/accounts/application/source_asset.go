package application

import money "github.com/pchkauu/want-keep/backend/internal/money/domain"

// CanonicalSourceAsset applies only mappings established by provider contracts.
// Callers retain the original external code as evidence and account metadata.
func CanonicalSourceAsset(provider, externalCode string) (money.Asset, error) {
	asset, err := money.ParseAsset(externalCode)
	if err != nil && externalCode == "RUR" && (provider == "raiffeisen" || provider == "ozon" || provider == "alfa") {
		return money.RUB, nil
	}
	return asset, err
}

// ValidateAsset applies only mappings established by the provider contracts; the raw code is retained unchanged.
func (input ImportInput) ValidateAsset() error {
	asset, err := CanonicalSourceAsset(input.Provider, input.ExternalAssetCode)
	if err != nil {
		return err
	}
	if asset != input.Asset {
		return money.ErrUnsupportedAsset
	}
	return nil
}
