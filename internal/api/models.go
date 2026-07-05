package api

// UserRegister defines the JSON schema for registration.
type UserRegister struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CarPredictionInput represents the single car prediction payload.
type CarPredictionInput struct {
	Year         int     `json:"year"`
	KmDriven     int     `json:"km_driven"`
	Fuel         string  `json:"fuel"`
	SellerType   string  `json:"seller_type"`
	Transmission string  `json:"transmission"`
	Owner        string  `json:"owner"`
	Mileage      float64 `json:"mileage"`
	Engine       float64 `json:"engine"`
	MaxPower     float64 `json:"max_power"`
	Seats        int     `json:"seats"`
}

// Validate checks the prediction input fields.
func (input *CarPredictionInput) Validate() []string {
	var errs []string
	if input.Year < 1990 || input.Year > 2026 {
		errs = append(errs, "year must be between 1990 and 2026")
	}
	if input.KmDriven < 0 {
		errs = append(errs, "km_driven must be greater than or equal to 0")
	}
	if input.Fuel != "Petrol" && input.Fuel != "Diesel" && input.Fuel != "CNG" && input.Fuel != "LPG" {
		errs = append(errs, "fuel must be one of Petrol, Diesel, CNG, LPG")
	}
	if input.SellerType != "Individual" && input.SellerType != "Dealer" && input.SellerType != "Trustmark Dealer" {
		errs = append(errs, "seller_type must be one of Individual, Dealer, Trustmark Dealer")
	}
	if input.Transmission != "Manual" && input.Transmission != "Automatic" {
		errs = append(errs, "transmission must be one of Manual, Automatic")
	}
	if input.Owner != "First Owner" && input.Owner != "Second Owner" && input.Owner != "Third Owner" && input.Owner != "Fourth & Above Owner" && input.Owner != "Test Drive Car" {
		errs = append(errs, "owner must be one of First Owner, Second Owner, Third Owner, Fourth & Above Owner, Test Drive Car")
	}
	if input.Mileage < 0 {
		errs = append(errs, "mileage must be greater than or equal to 0.0")
	}
	if input.Engine < 0 {
		errs = append(errs, "engine must be greater than or equal to 0.0")
	}
	if input.MaxPower < 0 {
		errs = append(errs, "max_power must be greater than or equal to 0.0")
	}
	if input.Seats < 2 || input.Seats > 10 {
		errs = append(errs, "seats must be between 2 and 10")
	}
	return errs
}
