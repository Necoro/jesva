package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"iter"
	"log"
	"math"
	"strconv"
	"strings"
)

type Cents int64

func (c Cents) Format(fmtStr string) string {
	eur, cts := c.AsEuro()
	return fmt.Sprintf(fmtStr, eur, cts)
}

func (c Cents) AsEuro() (int64, int64) {
	i := int64(c)
	return i / 100, i % 100
}

func (c Cents) String() string {
	return c.Format("%d.%02d EUR")
}

type SumType uint8

const (
	Amount SumType = iota
	Tax
	Ignore
)

type Eur struct {
	XmlName     xml.Name         `xml:"eur"`
	Name        string           `xml:"general>name"`
	Address     string           `xml:"general>address"`
	Company     string           `xml:"general>company"`
	TaxID       string           `xml:"general>taxid"`
	Start       Date             `xml:"general>businessyearrange>daterange>start>date"`
	End         Date             `xml:"general>businessyearrange>daterange>end>date"`
	Receipts    []*Receipt       `xml:"receipts>receipt"`
	Accounts    []Accounts       `xml:"accounts"`
	AccountInfo map[int]*Account `xml:"-"`
	TaxAccounts map[int]struct{} `xml:"-"`
}

type Date struct {
	Year  int `xml:"year,attr"`
	Month int `xml:"month,attr"`
	Day   int `xml:"day,attr"`
}

type Receipt struct {
	Number   int        `xml:"number"`
	Date     Date       `xml:"date"`
	Paid     bool       `xml:"paid,attr"`
	Payments []*Payment `xml:"payment"`
}

type Payment struct {
	Incoming int `xml:"taxaccountincoming"`
	Outgoing int `xml:"taxaccountoutgoing"`
	Account  int `xml:"account"`
	Amount   struct {
		TaxHandling string `xml:"tax,attr"`
		Value       string `xml:",chardata"`
		value       Cents
	} `xml:"amount"`
	receipt *Receipt // link back
}

type Accounts struct {
	Accounts []Account `xml:"account"`
	Type     string    `xml:"type,attr"`
}

type Account struct {
	Type       string `xml:"type,attr"`
	TaxAccount bool   `xml:"taxaccount,attr"`
	Rounding   string `xml:"rounding,attr"`
	Number     int    `xml:"number"`
	Percent    int    `xml:"percent"`
}

func (e *Eur) Year() int {
	return e.Start.Year
}

// prepareAccountInfo consolidates the information on tax accounts into
// the `AccountInfo` map
func (e *Eur) prepareAccountInfo() {
	e.AccountInfo = make(map[int]*Account)
	e.TaxAccounts = make(map[int]struct{})

	for _, a := range e.Accounts {
		switch a.Type {
		case "tax":
			for _, a := range a.Accounts {
				e.AccountInfo[a.Number] = &a
			}
		case "booking":
			for _, a := range a.Accounts {
				if a.TaxAccount {
					e.TaxAccounts[a.Number] = struct{}{}
				}
			}
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

	left, right, _ := strings.Cut(p.Amount.Value, ".")
	if len(right) < 2 {
		// trailing zeroes
		right = right + strings.Repeat("0", 2-len(right))
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
			if r.Paid && period.includes(r.Date) {
				for _, p := range r.Payments {
					_, isTaxAccount := e.TaxAccounts[p.Account]
					if (p.Incoming == account || p.Outgoing == account) && !isTaxAccount {
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
		var diff Cents
		switch sumType {
		case Amount:
			diff = p.getAmount(acc.Percent)
		case Tax:
			diff = p.getTax(acc.Percent)
		case Ignore:
			return 0
		default:
			log.Fatalf("Unexpected SumType: %v", sumType)
		}

		debug("Kto %02d/%02d (#%d):\t%s", account, p.Account, p.receipt.Number, diff.Format("%3d.%02d EUR"))
		sum += diff
	}

	return sum
}

func (e *Eur) Validate() {
	if e.Start.Year != e.End.Year {
		log.Fatalf("JES spans multiple years. This is not supported.")
	}

	// Check for unsupported tax accounts
	knownAccounts := make(map[int]struct{})
	for _, m := range mappings {
		knownAccounts[m.account] = struct{}{}
	}

	checkFn := func(acc int) {
		if acc == 0 { // account not given
			return
		}

		if _, ok := knownAccounts[acc]; !ok {
			log.Fatalf("Unsupported tax account '%d'", acc)
		}
	}

	for _, r := range e.Receipts {
		for _, p := range r.Payments {
			checkFn(p.Incoming)
			checkFn(p.Outgoing)
			p.receipt = r
		}
	}
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
