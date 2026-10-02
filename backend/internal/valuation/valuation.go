package valuation

import (
	"errors"
	"math/big"
	"strings"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

var ErrInvalidObservation = errors.New("invalid rate observation")
var ErrRateUnavailable = errors.New("rate unavailable")
var ErrQuotaExceeded = errors.New("rate source quota exceeded")
var ErrHistoryRange = errors.New("historical crypto price outside free range")

// Observation is one immutable provider value. A cross rate retains both source observations.
type Observation struct {
	ID, ProviderAssetID, Source, Transport string
	Revision                               uint64
	RequestedDate                          calendar.Date
	EffectiveAt, FetchedAt                 calendar.Instant
	Granularity                            string
	Rate                                   money.Rate
}

func (o Observation) Validate() error {
	if o.ID == "" || o.ProviderAssetID == "" || o.Source == "" || o.Transport == "" || o.Revision == 0 || o.RequestedDate.String() == "" || o.EffectiveAt.String() == "" || o.FetchedAt.String() == "" || (o.Granularity != "daily" && o.Granularity != "instant") || o.Rate.Validate() != nil {
		return ErrInvalidObservation
	}
	if o.Source == "cbr" {
		if o.ProviderAssetID != "R01235" || o.Rate.Base() != money.USD || o.Rate.Quote() != money.RUB || o.Granularity != "daily" || o.Transport != "xml_daily" && o.Transport != "frankfurter_v2" {
			return ErrInvalidObservation
		}
		return nil
	}
	ids := map[money.Asset]string{money.BTC: "bitcoin", money.ETH: "ethereum", money.USDT: "tether", money.USDC: "usd-coin"}
	if o.Source != "coingecko" || ids[o.Rate.Base()] != o.ProviderAssetID || o.Rate.Quote() != money.USD || o.Transport == "demo_history" && o.Granularity != "daily" || o.Transport == "demo_simple_price" && o.Granularity != "instant" || o.Transport != "demo_history" && o.Transport != "demo_simple_price" {
		return ErrInvalidObservation
	}
	return nil
}

// Convert preserves the native amount and calculates only a reporting equivalent.
// The supplied observations must be selected for the same requested date by the application.
func Convert(native money.Money, target money.Asset, observations []Observation) (money.Money, money.Rate, []Observation, error) {
	if err := native.Validate(); err != nil {
		return money.Money{}, money.Rate{}, nil, err
	}
	if _, err := money.ParseAsset(string(target)); err != nil {
		return money.Money{}, money.Rate{}, nil, err
	}
	if native.Asset() == target {
		return native, money.Rate{}, nil, nil
	}
	from, fromLeg, err := usdPrice(native.Asset(), observations)
	if err != nil {
		return money.Money{}, money.Rate{}, nil, err
	}
	to, toLeg, err := usdPrice(target, observations)
	if err != nil {
		return money.Money{}, money.Rate{}, nil, err
	}
	quotient := new(big.Rat).Quo(from, to)
	value, ok := new(big.Rat).SetString(native.Amount())
	if !ok {
		return money.Money{}, money.Rate{}, nil, money.ErrInvalidMoney
	}
	value.Mul(value, quotient)
	// This is a representation boundary. Source observations and native postings stay exact.
	converted, err := money.NewMoney(decimal(value, 80), target)
	if err != nil {
		return money.Money{}, money.Rate{}, nil, err
	}
	rate, err := money.NewRate(native.Asset(), target, decimal(quotient, 80))
	if err != nil {
		return money.Money{}, money.Rate{}, nil, err
	}
	legs := make([]Observation, 0, 2)
	if fromLeg != nil {
		legs = append(legs, *fromLeg)
	}
	if toLeg != nil {
		legs = append(legs, *toLeg)
	}
	return converted, rate, legs, nil
}

func usdPrice(asset money.Asset, observations []Observation) (*big.Rat, *Observation, error) {
	if asset == money.USD {
		return big.NewRat(1, 1), nil, nil
	}
	for i := range observations {
		o := &observations[i]
		if o.Validate() != nil {
			return nil, nil, ErrInvalidObservation
		}
		value, ok := new(big.Rat).SetString(o.Rate.Value())
		if !ok {
			return nil, nil, ErrInvalidObservation
		}
		if asset == money.RUB && o.Rate.Base() == money.USD && o.Rate.Quote() == money.RUB {
			return new(big.Rat).Inv(value), o, nil
		}
		if o.Rate.Base() == asset && o.Rate.Quote() == money.USD {
			return value, o, nil
		}
	}
	return nil, nil, ErrRateUnavailable
}

func decimal(value *big.Rat, scale int) string {
	negative := value.Sign() < 0
	numerator := new(big.Int).Abs(value.Num())
	numerator.Mul(numerator, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
	whole, remainder := new(big.Int).QuoRem(numerator, value.Denom(), new(big.Int))
	twice := new(big.Int).Lsh(remainder, 1)
	if twice.Cmp(value.Denom()) > 0 || (twice.Cmp(value.Denom()) == 0 && whole.Bit(0) == 1) {
		whole.Add(whole, big.NewInt(1))
	}
	digits := whole.String()
	if scale > 0 {
		if len(digits) <= scale {
			digits = strings.Repeat("0", scale-len(digits)+1) + digits
		}
		digits = digits[:len(digits)-scale] + "." + strings.TrimRight(digits[len(digits)-scale:], "0")
		digits = strings.TrimSuffix(digits, ".")
	}
	if negative && digits != "0" {
		return "-" + digits
	}
	return digits
}
