package geo

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeProvider struct {
	geocodeCalls int
	reverseCalls int
	geocodeOut   []GeocodeMatch
	reverseOut   ReverseResult
	geocodeErr   error
	reverseErr   error
}

func (f *fakeProvider) Name() string {
	return "fake"
}

func (f *fakeProvider) Geocode(_ context.Context, _ string, _ string, _ int) ([]GeocodeMatch, error) {
	f.geocodeCalls++
	if f.geocodeErr != nil {
		return nil, f.geocodeErr
	}
	return f.geocodeOut, nil
}

func (f *fakeProvider) Reverse(_ context.Context, _ float64, _ float64) (ReverseResult, error) {
	f.reverseCalls++
	if f.reverseErr != nil {
		return ReverseResult{}, f.reverseErr
	}
	return f.reverseOut, nil
}

func TestValidateAddressSuccess(t *testing.T) {
	provider := &fakeProvider{
		geocodeOut: []GeocodeMatch{
			{
				Latitude:    40.7484,
				Longitude:   -73.9857,
				DisplayName: "350 5th Avenue, New York, NY 10118, United States",
				Importance:  0.9,
				Address: AddressComponents{
					City:        "New York",
					State:       "NY",
					Country:     "United States",
					CountryCode: "US",
				},
			},
		},
	}
	service := NewService(provider, "US", 10*time.Minute)

	result, err := service.ValidateAddress(context.Background(), ValidateInput{
		Address: "  350   5th Ave,   New York  ",
	})
	if err != nil {
		t.Fatalf("validate address: %v", err)
	}

	if !result.IsValid {
		t.Fatalf("expected address to be valid")
	}
	if result.Region != "US" {
		t.Fatalf("expected region US, got %q", result.Region)
	}
	if result.NormalizedAddress != "350 5th Ave, New York" {
		t.Fatalf("unexpected normalized address: %q", result.NormalizedAddress)
	}
	if result.Standardized == "" {
		t.Fatalf("expected standardized address")
	}
	if provider.geocodeCalls != 1 {
		t.Fatalf("expected 1 geocode call, got %d", provider.geocodeCalls)
	}
}

func TestGeocodeUsesCache(t *testing.T) {
	provider := &fakeProvider{
		geocodeOut: []GeocodeMatch{
			{
				Latitude:    1.1,
				Longitude:   2.2,
				DisplayName: "result",
				Importance:  0.5,
			},
		},
	}
	service := NewService(provider, "", 10*time.Minute)

	_, err := service.Geocode(context.Background(), GeocodeInput{
		Address: "test address",
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("first geocode: %v", err)
	}

	_, err = service.Geocode(context.Background(), GeocodeInput{
		Address: "test address",
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("second geocode: %v", err)
	}

	if provider.geocodeCalls != 1 {
		t.Fatalf("expected cached geocode call count 1, got %d", provider.geocodeCalls)
	}
}

func TestReverseValidatesCoordinates(t *testing.T) {
	service := NewService(&fakeProvider{}, "", 10*time.Minute)

	_, err := service.Reverse(context.Background(), ReverseInput{
		Latitude:  200,
		Longitude: 10,
	})
	if err == nil {
		t.Fatalf("expected coordinate validation error")
	}
}

func TestReverseFallbackOnProviderFailure(t *testing.T) {
	service := NewService(&fakeProvider{reverseErr: errors.New("provider blocked")}, "", 10*time.Minute)

	result, err := service.Reverse(context.Background(), ReverseInput{
		Latitude:  40.7484,
		Longitude: -73.9857,
	})
	if err != nil {
		t.Fatalf("expected fallback result, got error: %v", err)
	}
	if result.Provider != "fallback" {
		t.Fatalf("expected fallback provider, got %q", result.Provider)
	}
	if result.Address.CountryCode != "US" {
		t.Fatalf("expected US fallback address, got %q", result.Address.CountryCode)
	}
}
