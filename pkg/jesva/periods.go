package jesva

import (
	"fmt"
	"strconv"
	"strings"
)

type Period interface {
	Includes(Date) bool
	String() string
}

func Parse(periodStr string) (Period, error) {
	if periodStr[0] == 'q' || periodStr[0] == 'Q' { // Quarter
		return ParseQuarter(periodStr)
	} else if startStr, endStr, found := strings.Cut(periodStr, "-"); found { // Month range
		return ParseMonths(startStr, endStr)
	} else if len(periodStr) <= 2 { // single month
		return ParseMonth(periodStr)
	} else if len(periodStr) == 4 { // year
		return ParseYear(periodStr)
	} else {
		return nil, fmt.Errorf("unknown period '%s'", periodStr)
	}
}

type Date interface {
	Year() int
	Month() int
	Day() int
}

type Month uint8

func (m Month) Includes(d Date) bool {
	return d.Month() == int(m)
}

func (m Month) String() string {
	return fmt.Sprintf("%02d", m)
}

func ParseMonth(str string) (Month, error) {
	month, err := strconv.ParseUint(str, 10, 8)
	if err != nil || month < 1 || month > 12 {
		return 0, fmt.Errorf("invalid month: %s (%w)", str, err)
	}
	return Month(month), nil
}

type Months struct {
	start Month
	end   Month
}

func (m Months) Includes(d Date) bool {
	return d.Month() >= int(m.start) && d.Month() <= int(m.end)
}

func (m Months) String() string {
	return fmt.Sprintf("%02d", m.end)
}

func ParseMonths(start, end string) (Months, error) {
	var err error
	var startM, endM Month

	if startM, err = ParseMonth(start); err != nil {
		return Months{}, fmt.Errorf("parsing start month: %w", err)
	}
	if endM, err = ParseMonth(end); err != nil {
		return Months{}, fmt.Errorf("parsing end month: %w", err)
	}

	if endM <= startM {
		return Months{}, fmt.Errorf("end month must be larger than starting month")
	}

	return Months{startM, endM}, nil
}

type Quarter uint8

const (
	Q1 Quarter = iota + 1
	Q2
	Q3
	Q4
)

func (q Quarter) Includes(d Date) bool {
	end := int(q) * 3
	return d.Month() <= end && d.Month() > end-3
}

func (q Quarter) String() string {
	// 4x = Qx
	return fmt.Sprintf("4%d", q)
}

func ParseQuarter(str string) (Quarter, error) {
	if len(str) != 2 || !(str[0] == 'Q' || str[0] == 'q') {
		return 0, fmt.Errorf("invalid quarter format: %s. Expected Qx, for x ∈ {1, 2, 3, 4}", str)
	}

	switch str[1] {
	case '1':
		return Q1, nil
	case '2':
		return Q2, nil
	case '3':
		return Q3, nil
	case '4':
		return Q4, nil
	}

	return 0, fmt.Errorf("unknown quarter '%s'", str)
}

type Year uint16

func (y Year) Includes(d Date) bool {
	return d.Year() == int(y)
}

func (y Year) String() string {
	return fmt.Sprintf("%d", y)
}

func ParseYear(str string) (Year, error) {
	year, err := strconv.ParseUint(str, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid year: %s (%w)", str, err)
	}
	return Year(year), nil
}
