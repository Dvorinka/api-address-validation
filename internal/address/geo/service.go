package geo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Service struct {
	provider      Provider
	defaultRegion string
	cacheTTL      time.Duration
	nowFn         func() time.Time

	mu            sync.RWMutex
	validateCache map[string]cachedValidate
	geocodeCache  map[string]cachedGeocode
	reverseCache  map[string]cachedReverse
}

type cachedValidate struct {
	result  ValidateResult
	expires time.Time
}

type cachedGeocode struct {
	result  []GeocodeMatch
	expires time.Time
}

type cachedReverse struct {
	result  ReverseResult
	expires time.Time
}

func NewService(provider Provider, defaultRegion string, cacheTTL time.Duration) *Service {
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	return &Service{
		provider:      provider,
		defaultRegion: normalizeRegion(defaultRegion),
		cacheTTL:      cacheTTL,
		nowFn:         func() time.Time { return time.Now().UTC() },
		validateCache: make(map[string]cachedValidate),
		geocodeCache:  make(map[string]cachedGeocode),
		reverseCache:  make(map[string]cachedReverse),
	}
}

func (s *Service) ValidateAddress(ctx context.Context, input ValidateInput) (ValidateResult, error) {
	normalized := normalizeAddress(input.Address)
	if normalized == "" {
		return ValidateResult{}, errors.New("address is required")
	}

	region := chooseRegion(input.Region, s.defaultRegion)
	cacheKey := strings.ToLower(normalized) + "|" + strings.ToUpper(region)
	if cached, ok := s.getValidateCache(cacheKey); ok {
		return cached, nil
	}

	matches, err := s.Geocode(ctx, GeocodeInput{
		Address: normalized,
		Region:  region,
		Limit:   1,
	})
	if err != nil {
		return ValidateResult{}, err
	}

	result := ValidateResult{
		InputAddress:      input.Address,
		NormalizedAddress: normalized,
		Region:            region,
		IsValid:           len(matches) > 0,
		Provider:          s.provider.Name(),
	}
	if len(matches) > 0 {
		best := matches[0]
		result.Standardized = best.DisplayName
		result.Latitude = best.Latitude
		result.Longitude = best.Longitude
		result.Address = best.Address
		result.Confidence = confidenceScore(best.Importance)
	}

	s.setValidateCache(cacheKey, result)
	return result, nil
}

func (s *Service) Geocode(ctx context.Context, input GeocodeInput) ([]GeocodeMatch, error) {
	address := normalizeAddress(input.Address)
	if address == "" {
		return nil, errors.New("address is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 1
	}
	if limit > 10 {
		limit = 10
	}

	region := chooseRegion(input.Region, s.defaultRegion)
	cacheKey := fmt.Sprintf("%s|%s|%d", strings.ToLower(address), strings.ToUpper(region), limit)
	if cached, ok := s.getGeocodeCache(cacheKey); ok {
		return cached, nil
	}

	matches, err := s.provider.Geocode(ctx, address, region, limit)
	if err != nil {
		fallback, ok := fallbackGeocode(address, region, limit)
		if !ok {
			return nil, err
		}
		s.setGeocodeCache(cacheKey, fallback)
		return fallback, nil
	}

	s.setGeocodeCache(cacheKey, matches)
	return matches, nil
}

func (s *Service) Reverse(ctx context.Context, input ReverseInput) (ReverseResult, error) {
	if err := validateCoordinates(input.Latitude, input.Longitude); err != nil {
		return ReverseResult{}, err
	}

	cacheKey := fmt.Sprintf("%.6f|%.6f", input.Latitude, input.Longitude)
	if cached, ok := s.getReverseCache(cacheKey); ok {
		return cached, nil
	}

	result, err := s.provider.Reverse(ctx, input.Latitude, input.Longitude)
	if err != nil {
		fallback, ok := fallbackReverse(input.Latitude, input.Longitude)
		if !ok {
			return ReverseResult{}, err
		}
		s.setReverseCache(cacheKey, fallback)
		return fallback, nil
	}
	if result.Provider == "" {
		result.Provider = s.provider.Name()
	}

	s.setReverseCache(cacheKey, result)
	return result, nil
}

func normalizeAddress(address string) string {
	parts := strings.Fields(strings.TrimSpace(address))
	return strings.Join(parts, " ")
}

func normalizeRegion(region string) string {
	region = strings.ToUpper(strings.TrimSpace(region))
	if len(region) > 2 {
		return ""
	}
	return region
}

func chooseRegion(requestRegion, defaultRegion string) string {
	region := normalizeRegion(requestRegion)
	if region != "" {
		return region
	}
	return defaultRegion
}

func validateCoordinates(latitude, longitude float64) error {
	if latitude < -90 || latitude > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if longitude < -180 || longitude > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
}

func confidenceScore(importance float64) float64 {
	if importance < 0 {
		return 0
	}
	if importance > 1 {
		return 1
	}
	return importance
}

func (s *Service) getValidateCache(key string) (ValidateResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.validateCache[key]
	if !ok || s.nowFn().After(item.expires) {
		return ValidateResult{}, false
	}
	return item.result, true
}

func (s *Service) setValidateCache(key string, value ValidateResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.validateCache[key] = cachedValidate{
		result:  value,
		expires: s.nowFn().Add(s.cacheTTL),
	}
}

func (s *Service) getGeocodeCache(key string) ([]GeocodeMatch, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.geocodeCache[key]
	if !ok || s.nowFn().After(item.expires) {
		return nil, false
	}
	out := make([]GeocodeMatch, len(item.result))
	copy(out, item.result)
	return out, true
}

func (s *Service) setGeocodeCache(key string, value []GeocodeMatch) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]GeocodeMatch, len(value))
	copy(out, value)
	s.geocodeCache[key] = cachedGeocode{
		result:  out,
		expires: s.nowFn().Add(s.cacheTTL),
	}
}

func (s *Service) getReverseCache(key string) (ReverseResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.reverseCache[key]
	if !ok || s.nowFn().After(item.expires) {
		return ReverseResult{}, false
	}
	return item.result, true
}

func (s *Service) setReverseCache(key string, value ReverseResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reverseCache[key] = cachedReverse{
		result:  value,
		expires: s.nowFn().Add(s.cacheTTL),
	}
}
