package geo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultNominatimBaseURL = "https://nominatim.openstreetmap.org"
	defaultUserAgent        = "apiservices-address-validation/1.0 (contact: dev@example.com)"
)

type NominatimProvider struct {
	baseURL   string
	userAgent string
	client    *http.Client
}

func NewNominatimProvider(baseURL, userAgent string, timeout time.Duration) *NominatimProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultNominatimBaseURL
	}
	if strings.TrimSpace(userAgent) == "" {
		userAgent = defaultUserAgent
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &NominatimProvider{
		baseURL:   strings.TrimRight(baseURL, "/"),
		userAgent: userAgent,
		client:    &http.Client{Timeout: timeout},
	}
}

func (p *NominatimProvider) Name() string {
	return "nominatim"
}

func (p *NominatimProvider) Geocode(ctx context.Context, address, region string, limit int) ([]GeocodeMatch, error) {
	if strings.TrimSpace(address) == "" {
		return nil, errors.New("address is required")
	}
	if limit <= 0 {
		limit = 1
	}
	if limit > 10 {
		limit = 10
	}

	values := url.Values{}
	values.Set("q", address)
	values.Set("format", "jsonv2")
	values.Set("addressdetails", "1")
	values.Set("limit", strconv.Itoa(limit))
	if region != "" {
		values.Set("countrycodes", strings.ToLower(region))
	}

	endpoint := p.baseURL + "/search?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", p.userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("geocode provider returned %d: %s", resp.StatusCode, string(raw))
	}

	var payload []struct {
		Lat         string  `json:"lat"`
		Lon         string  `json:"lon"`
		DisplayName string  `json:"display_name"`
		Importance  float64 `json:"importance"`
		Address     struct {
			HouseNumber   string `json:"house_number"`
			Road          string `json:"road"`
			Neighbourhood string `json:"neighbourhood"`
			Suburb        string `json:"suburb"`
			City          string `json:"city"`
			Town          string `json:"town"`
			Village       string `json:"village"`
			County        string `json:"county"`
			State         string `json:"state"`
			Postcode      string `json:"postcode"`
			Country       string `json:"country"`
			CountryCode   string `json:"country_code"`
		} `json:"address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]GeocodeMatch, 0, len(payload))
	for _, item := range payload {
		lat, err := strconv.ParseFloat(item.Lat, 64)
		if err != nil {
			continue
		}
		lon, err := strconv.ParseFloat(item.Lon, 64)
		if err != nil {
			continue
		}
		city := firstNonEmpty(item.Address.City, item.Address.Town, item.Address.Village)
		neighbourhood := firstNonEmpty(item.Address.Neighbourhood, item.Address.Suburb)
		out = append(out, GeocodeMatch{
			Latitude:    lat,
			Longitude:   lon,
			DisplayName: item.DisplayName,
			Importance:  item.Importance,
			Address: AddressComponents{
				HouseNumber:   item.Address.HouseNumber,
				Road:          item.Address.Road,
				Neighbourhood: neighbourhood,
				City:          city,
				County:        item.Address.County,
				State:         item.Address.State,
				PostalCode:    item.Address.Postcode,
				Country:       item.Address.Country,
				CountryCode:   strings.ToUpper(item.Address.CountryCode),
			},
		})
	}
	return out, nil
}

func (p *NominatimProvider) Reverse(ctx context.Context, latitude, longitude float64) (ReverseResult, error) {
	values := url.Values{}
	values.Set("lat", strconv.FormatFloat(latitude, 'f', 6, 64))
	values.Set("lon", strconv.FormatFloat(longitude, 'f', 6, 64))
	values.Set("format", "jsonv2")
	values.Set("addressdetails", "1")

	endpoint := p.baseURL + "/reverse?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ReverseResult{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", p.userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return ReverseResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return ReverseResult{}, fmt.Errorf("reverse provider returned %d: %s", resp.StatusCode, string(raw))
	}

	var payload struct {
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		DisplayName string `json:"display_name"`
		Address     struct {
			HouseNumber   string `json:"house_number"`
			Road          string `json:"road"`
			Neighbourhood string `json:"neighbourhood"`
			Suburb        string `json:"suburb"`
			City          string `json:"city"`
			Town          string `json:"town"`
			Village       string `json:"village"`
			County        string `json:"county"`
			State         string `json:"state"`
			Postcode      string `json:"postcode"`
			Country       string `json:"country"`
			CountryCode   string `json:"country_code"`
		} `json:"address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ReverseResult{}, err
	}

	lat, err := strconv.ParseFloat(payload.Lat, 64)
	if err != nil {
		return ReverseResult{}, errors.New("provider returned invalid latitude")
	}
	lon, err := strconv.ParseFloat(payload.Lon, 64)
	if err != nil {
		return ReverseResult{}, errors.New("provider returned invalid longitude")
	}

	city := firstNonEmpty(payload.Address.City, payload.Address.Town, payload.Address.Village)
	neighbourhood := firstNonEmpty(payload.Address.Neighbourhood, payload.Address.Suburb)
	return ReverseResult{
		Latitude:    lat,
		Longitude:   lon,
		DisplayName: payload.DisplayName,
		Address: AddressComponents{
			HouseNumber:   payload.Address.HouseNumber,
			Road:          payload.Address.Road,
			Neighbourhood: neighbourhood,
			City:          city,
			County:        payload.Address.County,
			State:         payload.Address.State,
			PostalCode:    payload.Address.Postcode,
			Country:       payload.Address.Country,
			CountryCode:   strings.ToUpper(payload.Address.CountryCode),
		},
		Provider: p.Name(),
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
