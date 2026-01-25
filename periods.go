package main

import (
	"fmt"
	"log"
	"strconv"
)

type Period interface {
	includes(Date) bool
	String() string
}

type Month uint8

func (m Month) includes(d Date) bool {
	return d.Month == int(m)
}

func (m Month) String() string {
	return fmt.Sprintf("%02d", m)
}

func parseMonth(str string) Month {
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

func (m Months) includes(d Date) bool {
	return d.Month >= int(m.start) && d.Month <= int(m.end)
}

func (m Months) String() string {
	return fmt.Sprintf("%02d", m.end)
}

type Quarter uint8

const (
	Q1 Quarter = iota + 1
	Q2
	Q3
	Q4
)

func (q Quarter) includes(d Date) bool {
	end := int(q) * 3
	return d.Month <= end && d.Month > end-3
}

func (q Quarter) String() string {
	// 4x = Qx
	return fmt.Sprintf("4%d", q)
}

type Year uint16

func (y Year) includes(d Date) bool {
	return d.Year == int(y)
}

func (y Year) String() string {
	return fmt.Sprintf("%d", y)
}

func parseYear(str string) Year {
	year, err := strconv.ParseUint(str, 10, 16)
	if err != nil {
		log.Fatalf("Invalid year: %s (%v)", str, err)
	}
	return Year(year)
}
