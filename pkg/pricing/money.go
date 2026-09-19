package pricing

import (
	"fmt"
	"strconv"
	"strings"
)

// Money represents a USD amount in micro-dollars (1 dollar = 1_000_000 micros).
// All cost arithmetic uses int64 to avoid floating-point precision loss.
type Money struct {
	Micros int64
}

// Add returns a + b.
func (m Money) Add(b Money) Money {
	return Money{Micros: m.Micros + b.Micros}
}

// Sub returns a - b.
func (m Money) Sub(b Money) Money {
	return Money{Micros: m.Micros - b.Micros}
}

// MulFloat returns m * q, where q is a float64 quantity (e.g. token count).
// The result is rounded to the nearest micro.
func (m Money) MulFloat(q float64) Money {
	return Money{Micros: int64(float64(m.Micros)*q + 0.5)}
}

// String returns the money as a decimal string with 6 decimal places.
func (m Money) String() string {
	whole := m.Micros / 1_000_000
	frac := m.Micros % 1_000_000
	if frac < 0 {
		frac = -frac
	}
	sign := ""
	if m.Micros < 0 {
		sign = "-"
		whole = -whole
	}
	return fmt.Sprintf("%s%d.%06d", sign, whole, frac)
}

// MarshalJSON encodes the money as a JSON string with 6 decimal places.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.String() + `"`), nil
}

// UnmarshalJSON parses a JSON string or number into Money.
func (m *Money) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		m.Micros = 0
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invalid money %q: %w", s, err)
	}
	m.Micros = int64(f * 1_000_000)
	return nil
}