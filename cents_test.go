package main

import (
	"testing"
)

func TestParseCents(t *testing.T) {
	tests := []struct {
		input   string
		want    Cents
		wantErr bool
	}{
		{"1.50", 150, false},
		{"1,50", 150, false},
		{"  1,50 ", 150, false},
		{"-0.01", -1, false},
		{" -0.01", -1, false},
		{"- 0.01", 0, true},
		{"150", 15000, false},
		{"1.5", 0, true},
		{"1.500", 0, true},
		{"abc", 0, true},
		{"-1.50", -150, false},
		{"1.-50", 0, true},
		{"-1.-50", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseCents(tt.input)
		if (err != nil) != tt.wantErr {
			t.Fatalf("ParseCents(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseCents(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		input Cents
		want  string
	}{
		{110, "1.10 EUR"},
		{1500, "15.00 EUR"},
		{30, "0.30 EUR"},
		{0, "0.00 EUR"},
		{-1, "-0.01 EUR"},
		{-2500, "-25.00 EUR"},
		{-130, "-1.30 EUR"},
		{-12, "-0.12 EUR"},
	}

	for _, tt := range tests {
		got := tt.input.String()
		if got != tt.want {
			t.Errorf("Cents(%d).String() = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEuroString(t *testing.T) {
	tests := []struct {
		input Cents
		want  string
	}{
		{10000, "100"},
		{10050, "100"},
		{-1, "0"},
		{0, "0"},
		{30, "0"},
		{-10000, "-100"},
		{-110, "-1"},
	}

	for _, tt := range tests {
		got := tt.input.EuroString()
		if got != tt.want {
			t.Errorf("Cents(%d).EuroString() = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFullEuros(t *testing.T) {
	tests := []struct {
		input Cents
		want  Cents
	}{
		{150, 100},
		{100, 100},
		{-150, -100},
		{-1, 0},
		{0, 0},
		{10, 0},
	}

	for _, tt := range tests {
		got := tt.input.FullEuros()
		if got != tt.want {
			t.Errorf("Cents(%d).FullEuros() = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNetAmount(t *testing.T) {
	tests := []struct {
		input Cents
		perc  int
		want  Cents
	}{
		{119, 19, 100},
		{120, 20, 100},
		{100, 0, 100},
		{-119, 19, -100},
	}

	for _, tt := range tests {
		got := tt.input.NetAmount(tt.perc)
		if got != tt.want {
			t.Errorf("Cents(%d).NetAmount(%d) = %v, want %v", tt.input, tt.perc, got, tt.want)
		}
	}
}

func TestPercentage(t *testing.T) {
	tests := []struct {
		input Cents
		perc  int
		want  Cents
	}{
		{100, 19, 19},
		{100, 20, 20},
		{100, 0, 0},
		{-100, 19, -19},
	}

	for _, tt := range tests {
		got := tt.input.Percentage(tt.perc)
		if got != tt.want {
			t.Errorf("Cents(%d).Percentage(%d) = %v, want %v", tt.input, tt.perc, got, tt.want)
		}
	}
}
