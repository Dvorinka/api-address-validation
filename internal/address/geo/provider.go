package geo

import "context"

type Provider interface {
	Name() string
	Geocode(ctx context.Context, address, region string, limit int) ([]GeocodeMatch, error)
	Reverse(ctx context.Context, latitude, longitude float64) (ReverseResult, error)
}
