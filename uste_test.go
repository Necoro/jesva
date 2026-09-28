package main

import (
	"bytes"
	"testing"
)

func TestUStELineString(t *testing.T) {
	tests := []struct {
		line UStELine
		want string
	}{
		{81, "81"},
		{5, "05"},
		{3701, "37/01"},
		{1205, "12/05"},
	}

	for _, tt := range tests {
		got := tt.line.String()
		if got != tt.want {
			t.Errorf("UStELine(%d).String() = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestPrintLine(t *testing.T) {
	rows := []struct {
		zeile    UStELine
		fullYear Kennzahl
		vz       Kennzahl
	}{
		// Amount without delta: informational tax in parentheses
		{81, Kennzahl{typ: Amount, percent: 19, amount: 100000}, Kennzahl{typ: Amount, percent: 19, amount: 100000}},
		// Amount with delta: tax delta in parentheses as well
		{3701, Kennzahl{typ: Amount, percent: 19, amount: 200000}, Kennzahl{typ: Amount, percent: 19, amount: 190000}},
		// Tax with delta: no parentheses, trailing space for alignment
		{66, Kennzahl{typ: Tax, withFraction: true, amount: 5000}, Kennzahl{typ: Tax, withFraction: true, amount: 4000}},
		// AmountOnly with delta on the amount
		{45, Kennzahl{typ: AmountOnly, amount: 30000}, Kennzahl{typ: AmountOnly, amount: 20000}},
		// AmountOnly without delta: only the amount
		{46, Kennzahl{typ: AmountOnly, amount: 10000}, Kennzahl{typ: AmountOnly, amount: 10000}},
	}

	var buf bytes.Buffer
	w := newTable(&buf)
	for _, r := range rows {
		printLine(w, &r.fullYear, &r.vz, r.zeile)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	want := "" +
		"     81  =>  1000,00 EUR  (190,00 EUR)             \n" +
		"  37/01  =>  2000,00 EUR  (380,00 EUR)  (19,00 EUR)\n" +
		"     66  =>    50,00 EUR                 10,00 EUR \n" +
		"     45  =>   300,00 EUR                100,00 EUR \n" +
		"     46  =>   100,00 EUR                           \n"

	got := buf.String()
	if got != want {
		t.Errorf("printLine output = %q, want %q", got, want)
	}
}
