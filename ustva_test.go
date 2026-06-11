package main

import (
	"bytes"
	"encoding/xml"
	"testing"
)

func TestAmountString(t *testing.T) {
	tests := []struct {
		withFraction bool
		amount       Cents
		want         string
	}{
		{false, 10000, "100"},
		{true, 10000, "100.00"},
		{false, -10000, "-100"},
		{true, -150, "-1.50"},
	}

	for _, tt := range tests {
		k := Kennzahl{withFraction: tt.withFraction, amount: tt.amount}
		got := k.amountString()
		if got != tt.want {
			t.Errorf("amountString(withFraction=%v, amount=%d) = %q, want %q",
				tt.withFraction, tt.amount, got, tt.want)
		}
	}
}

func TestMarshalXML(t *testing.T) {
	k := Kennzahlen{
		81: {withFraction: false, amount: 10020},
		66: {withFraction: true, amount: 1900},
	}

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	if err := k.MarshalXML(enc, xml.StartElement{}); err != nil {
		t.Fatalf("MarshalXML error: %v", err)
	}
	if err := enc.Flush(); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	want := "<Kz66>19.00</Kz66><Kz81>100</Kz81>"
	got := buf.String()
	if got != want {
		t.Errorf("MarshalXML output = %q, want %q", got, want)
	}
}
