package pricing

import (
	"encoding/json"
	"testing"
)

func TestMoney_Add(t *testing.T) {
	a := Money{Micros: 1_000_000} // $1.00
	b := Money{Micros: 250_000}   // $0.25
	got := a.Add(b)
	if got.Micros != 1_250_000 {
		t.Errorf("Add: got %d, want 1_250_000", got.Micros)
	}
}

func TestMoney_Sub(t *testing.T) {
	a := Money{Micros: 1_000_000}
	b := Money{Micros: 250_000}
	got := a.Sub(b)
	if got.Micros != 750_000 {
		t.Errorf("Sub: got %d, want 750_000", got.Micros)
	}
}

func TestMoney_MulFloat(t *testing.T) {
	// $0.000003 per token, 1500 tokens → $0.0045
	perToken := Money{Micros: 3}
	got := perToken.MulFloat(1500)
	if got.Micros != 4_500 {
		t.Errorf("MulFloat: got %d, want 4_500", got.Micros)
	}
}

func TestMoney_String(t *testing.T) {
	cases := []struct {
		in   Money
		want string
	}{
		{Money{Micros: 1_000_000}, "1.000000"},
		{Money{Micros: 250_000}, "0.250000"},
		{Money{Micros: 0}, "0.000000"},
		{Money{Micros: 1}, "0.000001"},
		{Money{Micros: 123_456_789}, "123.456789"},
	}
	for _, tc := range cases {
		if got := tc.in.String(); got != tc.want {
			t.Errorf("String(%d): got %q, want %q", tc.in.Micros, got, tc.want)
		}
	}
}

func TestMoney_JSONRoundtrip(t *testing.T) {
	cases := []Money{
		{Micros: 0},
		{Micros: 1},
		{Micros: 1_000_000},
		{Micros: 123_456_789},
	}
	for _, m := range cases {
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var got Money
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if got.Micros != m.Micros {
			t.Errorf("roundtrip: got %d, want %d", got.Micros, m.Micros)
		}
	}
}

func TestMoney_JSONMarshalFormat(t *testing.T) {
	m := Money{Micros: 1_500_000}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `"1.500000"`
	if string(b) != want {
		t.Errorf("Marshal: got %s, want %s", string(b), want)
	}
}