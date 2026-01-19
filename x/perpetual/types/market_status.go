package types

// MarketStatus represents the status of a market
type MarketStatus int

const (
	MarketStatusInactive MarketStatus = iota // Market is inactive
	MarketStatusActive                       // Market is active and trading
	MarketStatusSettling                     // Market is settling funding rate
	MarketStatusPaused                       // Market is paused (no new orders)
)

// String returns the string representation of MarketStatus
func (s MarketStatus) String() string {
	switch s {
	case MarketStatusActive:
		return "active"
	case MarketStatusSettling:
		return "settling"
	case MarketStatusPaused:
		return "paused"
	default:
		return "inactive"
	}
}

// IsActive returns true if the market accepts new orders
func (s MarketStatus) IsActive() bool {
	return s == MarketStatusActive
}

// IsTradeable returns true if the market can execute trades
func (s MarketStatus) IsTradeable() bool {
	return s == MarketStatusActive || s == MarketStatusSettling
}
