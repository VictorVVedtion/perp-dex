package types

// MarginMode represents the margin mode for an account
type MarginMode int

const (
	MarginModeIsolated MarginMode = iota // Isolated margin mode (per-position)
	MarginModeCross                      // Cross margin mode (shared across positions)
)

// String returns the string representation of MarginMode
func (m MarginMode) String() string {
	switch m {
	case MarginModeCross:
		return "cross"
	default:
		return "isolated"
	}
}

// IsCross returns true if the margin mode is cross
func (m MarginMode) IsCross() bool {
	return m == MarginModeCross
}

// IsIsolated returns true if the margin mode is isolated
func (m MarginMode) IsIsolated() bool {
	return m == MarginModeIsolated
}
