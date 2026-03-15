package geo

type AddressComponents struct {
	HouseNumber   string `json:"house_number,omitempty"`
	Road          string `json:"road,omitempty"`
	Neighbourhood string `json:"neighbourhood,omitempty"`
	City          string `json:"city,omitempty"`
	County        string `json:"county,omitempty"`
	State         string `json:"state,omitempty"`
	PostalCode    string `json:"postal_code,omitempty"`
	Country       string `json:"country,omitempty"`
	CountryCode   string `json:"country_code,omitempty"`
}

type GeocodeMatch struct {
	Latitude    float64           `json:"latitude"`
	Longitude   float64           `json:"longitude"`
	DisplayName string            `json:"display_name"`
	Importance  float64           `json:"importance"`
	Address     AddressComponents `json:"address"`
}

type ValidateInput struct {
	Address string `json:"address"`
	Region  string `json:"region,omitempty"`
}

type GeocodeInput struct {
	Address string `json:"address"`
	Region  string `json:"region,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type ReverseInput struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type ValidateResult struct {
	InputAddress      string            `json:"input_address"`
	NormalizedAddress string            `json:"normalized_address"`
	Region            string            `json:"region,omitempty"`
	IsValid           bool              `json:"is_valid"`
	Standardized      string            `json:"standardized_address,omitempty"`
	Latitude          float64           `json:"latitude,omitempty"`
	Longitude         float64           `json:"longitude,omitempty"`
	Confidence        float64           `json:"confidence,omitempty"`
	Address           AddressComponents `json:"address,omitempty"`
	Provider          string            `json:"provider"`
}

type ReverseResult struct {
	Latitude    float64           `json:"latitude"`
	Longitude   float64           `json:"longitude"`
	DisplayName string            `json:"display_name"`
	Address     AddressComponents `json:"address"`
	Provider    string            `json:"provider"`
}
