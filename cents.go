package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Cents int64

func (c Cents) Format(fmtStr string) string {
	cInt := int64(c)
	prefix := ""
	if cInt < 0 {
		prefix = "-"
		cInt = -cInt
	}

	return prefix + fmt.Sprintf(fmtStr, cInt/100, cInt%100)
}

func (c Cents) Cents() Cents {
	return c % 100
}

func (c Cents) FullEuros() Cents {
	return c - c.Cents()
}

func (c Cents) String() string {
	return c.Format("%d.%02d EUR")
}

// EuroString formats the amount in full euros.
func (c Cents) EuroString() string {
	return strconv.FormatInt(int64(c)/100, 10)
}

// NetAmount returns the net amount under the given percentage.
// The percentage is specified in percentage notation (i.e. 20 for 20%).
func (c Cents) NetAmount(perc int) Cents {
	if perc == 0 {
		return c
	}

	factor := float64(perc)/100 + 1
	return Cents(math.Round(float64(c) / factor))
}

// Percentage returns the given percentage, where perc is specified in percentage notation (i.e. 20 for 20%).
func (c Cents) Percentage(perc int) Cents {
	if perc == 0 {
		return 0
	}

	factor := float64(perc) / 100
	return Cents(math.Round(float64(c) * factor))
}

func parseCents(eurStr, centsStr string) (Cents, error) {
	var eur, cents int
	var err error

	if eur, err = strconv.Atoi(eurStr); err != nil {
		return 0, fmt.Errorf("parsing euro: %w", err)
	}
	if centsStr != "" {
		if cents, err = strconv.Atoi(centsStr); err != nil {
			return 0, fmt.Errorf("parsing cents: %w", err)
		}
	}

	if eur < 0 || (eur == 0 && eurStr[0] == '-') {
		cents = -cents
	}
	return Cents(eur*100 + cents), nil
}

func ParseCents(str string) (Cents, error) {
	str = strings.TrimSpace(str)
	if eurStr, centsStr, found := strings.Cut(str, "."); found && len(centsStr) == 2 {
		return parseCents(eurStr, centsStr)
	}
	if eurStr, centsStr, found := strings.Cut(str, ","); found && len(centsStr) == 2 {
		return parseCents(eurStr, centsStr)
	}
	if !strings.ContainsAny(str, ",.") {
		return parseCents(str, "")
	}
	return 0, fmt.Errorf("invalid format: '%s'", str)
}
