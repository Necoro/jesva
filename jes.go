package main

import (
	"archive/zip"
	"encoding/xml"
	"iter"
	"log"
	"math"
	"strconv"
	"strings"
)

type Cents = int64

type SumType bool

const (
	Amount SumType = true
	Tax    SumType = false
)

type Eur struct {
	XmlName     xml.Name         `xml:"eur"`
	Name        string           `xml:"general>name"`
	Address     string           `xml:"general>address"`
	Company     string           `xml:"general>company"`
	TaxID       string           `xml:"general>taxid"`
	Start       Date             `xml:"general>businessyearrange>daterange>start>date"`
	Receipts    []Receipt        `xml:"receipts>receipt"`
	Accounts    []Accounts       `xml:"accounts"`
	AccountInfo map[int]*Account `xml:"-"`
}

type Date struct {
	Year  int `xml:"year,attr"`
	Month int `xml:"month,attr"`
	Day   int `xml:"day,attr"`
}

type Receipt struct {
	Date     Date       `xml:"date"`
	Payments []*Payment `xml:"payment"`
}

type Payment struct {
	Incoming int `xml:"taxaccountincoming"`
	Outgoing int `xml:"taxaccountoutgoing"`
	Amount   struct {
		TaxHandling string `xml:"tax,attr"`
		Value       string `xml:",chardata"`
		value       Cents
	} `xml:"amount"`
}

type Accounts struct {
	Accounts []Account `xml:"account"`
	Type     string    `xml:"type,attr"`
}

type Account struct {
	Type     string `xml:"type,attr"`
	Rounding string `xml:"rounding,attr"`
	Number   int    `xml:"number"`
	Percent  int    `xml:"percent"`
}

func (e *Eur) Year() int {
	return e.Start.Year
}

// prepareAccountInfo consolidates the information on tax accounts into
// the `AccountInfo` map
func (e *Eur) prepareAccountInfo() {
	e.AccountInfo = make(map[int]*Account)

	for _, a := range e.Accounts {
		if a.Type != "tax" {
			continue
		}

		for _, a := range a.Accounts {
			e.AccountInfo[a.Number] = &a
		}
	}
}

func (p *Payment) isIncludingTax() bool {
	return p.Amount.TaxHandling == "incl"
}

// getValue returns the value of that Receipt in Cents.
func (p *Payment) getValue() Cents {
	if p.Amount.value != 0 {
		return p.Amount.value
	}

	left, right, found := strings.Cut(p.Amount.Value, ".")
	if !found {
		right = "00"
	}

	ival, err := strconv.Atoi(left + right)
	if err != nil {
		log.Fatalf("Problem parsing amount '%s%s' as int: %v", left, right, err)
	}

	p.Amount.value = Cents(ival)
	return p.Amount.value
}

// getAmount returns the gross amount of this Payment, i.e. without taxes.
func (p *Payment) getAmount(perc int) Cents {
	val := p.getValue()

	if !p.isIncludingTax() || perc == 0 {
		return val
	}

	factor := float64(perc)/100 + 1
	amt := math.Round(float64(val) / factor)
	return Cents(amt)
}

// getTax returns the taxes of this Payment.
func (p *Payment) getTax(perc int) Cents {
	if perc == 0 {
		return Cents(0)
	}

	val := p.getValue()

	if p.isIncludingTax() {
		amt := p.getAmount(perc)
		return val - amt
	}

	factor := float64(perc) / 100
	tax := math.Round(float64(val) * factor)
	return Cents(tax)
}

func (e *Eur) payments(account int, period Period) iter.Seq[*Payment] {
	return func(yield func(*Payment) bool) {
		for _, r := range e.Receipts {
			if period.includes(r.Date) {
				for _, p := range r.Payments {
					if p.Incoming == account || p.Outgoing == account {
						if !yield(p) {
							return
						}
					}
				}
			}
		}
	}
}

// ReceiptSum gathers the sum of all relevant receipts for that account.
func (e *Eur) ReceiptSum(account int, sumType SumType, period Period) Cents {
	acc := e.AccountInfo[account]
	if acc == nil {
		log.Fatalf("No info found for account %d", account)
	}

	var sum Cents
	for p := range e.payments(account, period) {
		switch sumType {
		case Amount:
			sum += p.getAmount(acc.Percent)
		case Tax:
			sum += p.getTax(acc.Percent)
		default:
			log.Fatalf("Unexpected SumType: %v", sumType)
		}
	}

	return sum
}

func readJesFile(jesFile string) *Eur {
	// JES stores the file as a ZIP archive
	zipF, err := zip.OpenReader(jesFile)
	if err != nil {
		log.Fatalf("Opening file '%s': %v", jesFile, err)
	}
	defer zipF.Close()

	// data is stored in `data.xml`
	var dataFile *zip.File
	for _, file := range zipF.File {
		if file.Name == "data.xml" {
			dataFile = file
			break
		}
	}

	if dataFile == nil {
		log.Fatalf("Could not find data.xml in JES-File '%s'.", jesFile)
	}

	f, err := dataFile.Open()
	if err != nil {
		log.Fatalf("Decompressing file '%s': %v", jesFile, err)
	}
	defer f.Close()

	// Decode XML data
	eur := new(Eur)
	decoder := xml.NewDecoder(f)
	if err = decoder.Decode(eur); err != nil {
		log.Fatalf("Decoding '%s': %v", jesFile, err)
	}

	eur.prepareAccountInfo()

	return eur
}
