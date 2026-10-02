package rates

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
	"golang.org/x/text/encoding/charmap"
)

var (
	ErrUnavailable   = errors.New("rate source unavailable")
	ErrRateLimited   = errors.New("rate source rate limited")
	ErrConfiguration = errors.New("rate source configuration invalid")
	ErrHistoryRange  = valuation.ErrHistoryRange
)

const (
	cbrURL         = "https://www.cbr.ru/scripts/XML_daily.asp"
	frankfurterURL = "https://api.frankfurter.dev/v2/rate/usd/rub"
	coinGeckoURL   = "https://api.coingecko.com/api/v3"
)

var cryptoIDs = map[money.Asset]string{
	money.BTC: "bitcoin", money.ETH: "ethereum",
	money.USDT: "tether", money.USDC: "usd-coin",
}

// Client reads reference rates only. Endpoint overrides are unexported test seams.
type Client struct {
	HTTP             *http.Client
	CoinGeckoKeyFile string
	Quota            interface {
		ReserveRateCall(context.Context, time.Time) error
	}
	cbr, frankfurter, cg string
}

func New(client *http.Client, keyFile string, quota interface {
	ReserveRateCall(context.Context, time.Time) error
}) *Client {
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &Client{HTTP: client, CoinGeckoKeyFile: keyFile, Quota: quota, cbr: cbrURL, frankfurter: frankfurterURL, cg: coinGeckoURL}
}

func (c *Client) USDToRUB(ctx context.Context, date calendar.Date, fetched time.Time) (valuation.Observation, error) {
	primary, err := c.fetchCBR(ctx, date, fetched)
	if err == nil {
		return primary, nil
	}
	fallback, fallbackErr := c.fetchFrankfurter(ctx, date, fetched)
	if fallbackErr == nil {
		return fallback, nil
	}
	return valuation.Observation{}, errors.Join(err, fallbackErr)
}

func (c *Client) USDToRUBCandidates(ctx context.Context, date calendar.Date, fetched time.Time) ([]valuation.Observation, error) {
	primary, primaryErr := c.fetchCBR(ctx, date, fetched)
	fallback, fallbackErr := c.fetchFrankfurter(ctx, date, fetched)
	result := []valuation.Observation{}
	if primaryErr == nil {
		result = append(result, primary)
	}
	if fallbackErr == nil {
		result = append(result, fallback)
	}
	if len(result) == 0 {
		return nil, errors.Join(primaryErr, fallbackErr)
	}
	return result, nil
}

func (c *Client) CryptoUSD(ctx context.Context, asset money.Asset, date calendar.Date, fetched time.Time) (valuation.Observation, error) {
	id, ok := cryptoIDs[asset]
	if !ok || date.String() == "" {
		return valuation.Observation{}, ErrUnavailable
	}
	day := fetched.UTC().Truncate(24 * time.Hour)
	if dateTime(date).After(day) || day.Sub(dateTime(date)) > 365*24*time.Hour {
		return valuation.Observation{}, ErrHistoryRange
	}
	key, err := c.coinGeckoKey()
	if err != nil {
		return valuation.Observation{}, err
	}
	path := c.cg + "/coins/" + id + "/history"
	query := url.Values{"date": {dateTime(date).Format("02-01-2006")}, "localization": {"false"}}
	body, err := c.get(ctx, path+"?"+query.Encode(), key)
	if err != nil {
		return valuation.Observation{}, err
	}
	var response struct {
		MarketData struct {
			CurrentPrice map[string]json.Number `json:"current_price"`
		} `json:"market_data"`
	}
	if err = decode(body, &response); err != nil {
		return valuation.Observation{}, err
	}
	// CoinGecko history is a daily UTC snapshot, not an executable timestamp.
	return point(asset, id, "coingecko", "demo_history", date, dateTime(date), fetched, "daily", response.MarketData.CurrentPrice["usd"].String())
}

func (c *Client) CryptoCurrent(ctx context.Context, fetched time.Time) ([]valuation.Observation, error) {
	key, err := c.coinGeckoKey()
	if err != nil {
		return nil, err
	}
	query := url.Values{"ids": {"bitcoin,ethereum,tether,usd-coin"}, "vs_currencies": {"usd"}, "include_last_updated_at": {"true"}}
	body, err := c.get(ctx, c.cg+"/simple/price?"+query.Encode(), key)
	if err != nil {
		return nil, err
	}
	var response map[string]struct {
		USD           json.Number `json:"usd"`
		LastUpdatedAt json.Number `json:"last_updated_at"`
	}
	if err = decode(body, &response); err != nil {
		return nil, err
	}
	out := make([]valuation.Observation, 0, len(cryptoIDs))
	for _, asset := range []money.Asset{money.BTC, money.ETH, money.USDT, money.USDC} {
		id := cryptoIDs[asset]
		value, ok := response[id]
		if !ok {
			return nil, ErrUnavailable
		}
		seconds, parseErr := value.LastUpdatedAt.Int64()
		if parseErr != nil || seconds <= 0 || time.Unix(seconds, 0).After(fetched.Add(time.Minute)) {
			return nil, ErrUnavailable
		}
		at := time.Unix(seconds, 0).UTC()
		date, _ := calendar.ParseDate(at.Format(time.DateOnly))
		p, pointErr := point(asset, id, "coingecko", "demo_simple_price", date, at, fetched, "instant", value.USD.String())
		if pointErr != nil {
			return nil, pointErr
		}
		out = append(out, p)
	}
	return out, nil
}

func (c *Client) fetchCBR(ctx context.Context, date calendar.Date, fetched time.Time) (valuation.Observation, error) {
	if date.String() == "" {
		return valuation.Observation{}, ErrUnavailable
	}
	query := url.Values{"date_req": {dateTime(date).Format("02/01/2006")}}
	body, err := c.get(ctx, c.cbr+"?"+query.Encode(), "")
	if err != nil {
		return valuation.Observation{}, err
	}
	var response struct {
		Date   string `xml:"Date,attr"`
		Valute []struct {
			ID      string `xml:"ID,attr"`
			Nominal string `xml:"Nominal"`
			Value   string `xml:"Value"`
		} `xml:"Valute"`
	}
	decoder := xml.NewDecoder(strings.NewReader(string(body)))
	decoder.CharsetReader = func(label string, input io.Reader) (io.Reader, error) {
		if !strings.EqualFold(label, "windows-1251") {
			return nil, ErrUnavailable
		}
		return charmap.Windows1251.NewDecoder().Reader(input), nil
	}
	if err = decoder.Decode(&response); err != nil {
		return valuation.Observation{}, ErrUnavailable
	}
	effective, err := time.Parse("02.01.2006", response.Date)
	if err != nil || effective.After(dateTime(date)) {
		return valuation.Observation{}, ErrUnavailable
	}
	for _, v := range response.Valute {
		if v.ID != "R01235" {
			continue
		}
		n := new(big.Rat)
		value := new(big.Rat)
		if _, ok := n.SetString(v.Nominal); !ok || n.Sign() <= 0 {
			return valuation.Observation{}, ErrUnavailable
		}
		if _, ok := value.SetString(strings.ReplaceAll(v.Value, ",", ".")); !ok {
			return valuation.Observation{}, ErrUnavailable
		}
		rate := new(big.Rat).Quo(value, n)
		return point(money.RUB, v.ID, "cbr", "xml_daily", date, effective, fetched, "daily", rate.FloatString(30))
	}
	return valuation.Observation{}, ErrUnavailable
}

func (c *Client) fetchFrankfurter(ctx context.Context, date calendar.Date, fetched time.Time) (valuation.Observation, error) {
	query := url.Values{"date": {date.String()}, "providers": {"cbr"}}
	body, err := c.get(ctx, c.frankfurter+"?"+query.Encode(), "")
	if err != nil {
		return valuation.Observation{}, err
	}
	var response struct {
		Date  string      `json:"date"`
		Base  string      `json:"base"`
		Quote string      `json:"quote"`
		Rate  json.Number `json:"rate"`
	}
	if err = decode(body, &response); err != nil || response.Base != "USD" || response.Quote != "RUB" {
		return valuation.Observation{}, ErrUnavailable
	}
	effective, err := calendar.ParseDate(response.Date)
	if err != nil || effective.String() > date.String() {
		return valuation.Observation{}, ErrUnavailable
	}
	return point(money.RUB, "R01235", "cbr", "frankfurter_v2", date, dateTime(effective), fetched, "daily", response.Rate.String())
}

func point(asset money.Asset, providerID, source, transport string, date calendar.Date, effective, fetched time.Time, granularity, value string) (valuation.Observation, error) {
	base, quote := asset, money.USD
	if asset == money.RUB {
		base, quote = money.USD, money.RUB
	}
	decimal, err := plainDecimal(value)
	if err != nil {
		return valuation.Observation{}, ErrUnavailable
	}
	rate, err := money.NewRate(base, quote, decimal)
	if err != nil {
		return valuation.Observation{}, ErrUnavailable
	}
	at, err := calendar.ParseInstant(effective.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return valuation.Observation{}, err
	}
	got, err := calendar.ParseInstant(fetched.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return valuation.Observation{}, err
	}
	out := valuation.Observation{ID: uuid.NewString(), ProviderAssetID: providerID, Source: source, Transport: transport, Revision: 1, RequestedDate: date, EffectiveAt: at, FetchedAt: got, Granularity: granularity, Rate: rate}
	return out, out.Validate()
}

func plainDecimal(value string) (string, error) {
	if len(value) == 0 || len(value) > 256 {
		return "", ErrUnavailable
	}
	parts := strings.Split(strings.ToLower(value), "e")
	if len(parts) == 1 {
		return value, nil
	}
	if len(parts) != 2 {
		return "", ErrUnavailable
	}
	exponent, err := strconv.Atoi(parts[1])
	if err != nil || exponent < -256 || exponent > 256 {
		return "", ErrUnavailable
	}
	mantissa := parts[0]
	sign := ""
	if strings.HasPrefix(mantissa, "-") {
		sign, mantissa = "-", mantissa[1:]
	}
	dot := strings.IndexByte(mantissa, '.')
	if dot < 0 {
		dot = len(mantissa)
	}
	digits := strings.ReplaceAll(mantissa, ".", "")
	position := dot + exponent
	var decimal string
	switch {
	case position <= 0:
		decimal = "0." + strings.Repeat("0", -position) + digits
	case position >= len(digits):
		decimal = digits + strings.Repeat("0", position-len(digits))
	default:
		decimal = digits[:position] + "." + digits[position:]
	}
	decimal = sign + decimal
	if len(decimal) > 256 {
		return "", ErrUnavailable
	}
	return decimal, nil
}

func (c *Client) coinGeckoKey() (string, error) {
	if c.CoinGeckoKeyFile == "" {
		return "", ErrConfiguration
	}
	info, err := os.Stat(c.CoinGeckoKeyFile)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 4096 {
		return "", ErrConfiguration
	}
	contents, err := os.ReadFile(c.CoinGeckoKeyFile)
	if err != nil {
		return "", ErrConfiguration
	}
	key := strings.TrimSpace(string(contents))
	if key == "" || strings.ContainsAny(key, "\r\n\t ") {
		return "", ErrConfiguration
	}
	return key, nil
}

func (c *Client) get(ctx context.Context, address, key string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, ErrConfiguration
	}
	if key != "" {
		if c.Quota == nil {
			return nil, ErrConfiguration
		}
		if err := c.Quota.ReserveRateCall(ctx, time.Now().UTC()); err != nil {
			if errors.Is(err, valuation.ErrQuotaExceeded) {
				return nil, ErrRateLimited
			}
			return nil, ErrUnavailable
		}
		req.Header.Set("x-cg-demo-api-key", key)
	}
	client := *c.HTTP
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	res, err := client.Do(req)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrConfiguration
	default:
		return nil, ErrUnavailable
	}
	if res.ContentLength > 1<<20 {
		return nil, ErrUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return nil, ErrUnavailable
	}
	return body, nil
}

func decode(body []byte, out any) error {
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("%w: invalid provider response", ErrUnavailable)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return ErrUnavailable
	}
	return nil
}

func dateTime(date calendar.Date) time.Time {
	t, _ := time.Parse(time.DateOnly, date.String())
	return t.UTC()
}
