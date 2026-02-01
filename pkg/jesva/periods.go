package jesva

import (
	"fmt"
	"log"
	"strconv"
)

type Period interface {
	Includes(Date) bool
	String() string
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

func ParseMonth(str string) Month {
	month, err := strconv.ParseUint(str, 10, 8)
	if err != nil || month < 1 || month > 12 {
		log.Fatalf("Invalid month: %s (%v)", str, err)
	}
	return Month(month)
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

func ParseMonths(start, end string) Months {
	startM := ParseMonth(start)
	endM := ParseMonth(end)

	if endM <= startM {
		log.Fatalf("End month must be larger than starting month.")
	}

	return Months{startM, endM}
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

func ParseQuarter(str string) Quarter {
	if len(str) != 2 || !(str[0] == 'Q' || str[0] == 'q') {
		log.Fatalf("Invalid quarter format: %s. Expected Qx, for x ∈ {1, 2, 3, 4}", str)
	}

	switch str[1] {
	case '1':
		return Q1
	case '2':
		return Q2
	case '3':
		return Q3
	case '4':
		return Q4
	}

	log.Fatalf("Unknown quarter '%s'", str)
	return Q1 // unreachable
}

type Year uint16

func (y Year) Includes(d Date) bool {
	return d.Year() == int(y)
}

func (y Year) String() string {
	return fmt.Sprintf("%d", y)
}

func ParseYear(str string) Year {
	year, err := strconv.ParseUint(str, 10, 16)
	if err != nil {
		log.Fatalf("Invalid year: %s (%v)", str, err)
	}
	return Year(year)
}
