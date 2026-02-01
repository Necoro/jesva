package jes

import (
	"archive/zip"
	"cmp"
	"encoding/xml"
	"iter"
	"log"
	"slices"
	"strconv"
	"strings"

	"github.com/Necoro/jesva/pkg/jesva"
)

type Eur struct {
	XmlName            xml.Name   `xml:"eur"`
	Name               string     `xml:"general>name"`
	Address            string     `xml:"general>address"`
	Company            string     `xml:"general>company"`
	TaxID              string     `xml:"general>taxid"`
	Start              Date       `xml:"general>businessyearrange>daterange>start>date"`
	End                Date       `xml:"general>businessyearrange>daterange>end>date"`
	Receipts           []*Receipt `xml:"receipts>receipt"`
	Accounts           []Accounts `xml:"accounts"`
	accountInfo        map[TaxAccount]Account
	taxBookingAccounts map[int]struct{} // accounts where taxes are booked as revenue, e.g. paid taxes
}

type Date struct {
	Y int `xml:"year,attr"`
	M int `xml:"month,attr"`
	D int `xml:"day,attr"`
}

func (d Date) Year() int {
	return d.Y
}

func (d Date) Month() int {
	return d.M
}

func (d Date) Day() int {
	return d.D
}

type Receipt struct {
	Number   int        `xml:"number"`
	Date     Date       `xml:"date"`
	Paid     bool       `xml:"paid,attr"`
	Payments []*Payment `xml:"payment"`
}

type Payment struct {
	Incoming TaxAccount `xml:"taxaccountincoming"`
	Outgoing TaxAccount `xml:"taxaccountoutgoing"`
	Account  int        `xml:"account"`
	Amount   struct {
		TaxHandling string `xml:"tax,attr"`
		Value       string `xml:",chardata"`
		value       jesva.Cents
	} `xml:"amount"`
	receipt *Receipt // link back
}

type Accounts struct {
	Accounts []Account `xml:"account"`
	Type     string    `xml:"type,attr"`
}

type Account struct {
	TaxAccount bool `xml:"taxaccount,attr"`
	Number     int  `xml:"number"`
	Percent    int  `xml:"percent"`
}

type TaxAccount uint16

// minIncomeAccount is the lowest tax account number that is considered an expense.
// All accounts below this number are considered expense.
const minIncomeAccount = 500

func (t TaxAccount) IsExpense() bool {
	return !t.IsIncome()
}

func (t TaxAccount) IsIncome() bool {
	return t >= minIncomeAccount
}

func (e *Eur) AccountFor(t TaxAccount) Account {
	return e.accountInfo[t]
}

func (e *Eur) Year() int {
	return e.Start.Year()
}

// prepareAccountInfo consolidates the information on tax accounts into
// the `accountInfo` map
func (e *Eur) prepareAccountInfo() {
	e.accountInfo = make(map[TaxAccount]Account)
	e.taxBookingAccounts = make(map[int]struct{})

	for _, a := range e.Accounts {
		switch a.Type {
		case "tax":
			for _, a := range a.Accounts {
				e.accountInfo[TaxAccount(a.Number)] = a
			}
		case "booking":
			for _, a := range a.Accounts {
				if a.TaxAccount {
					e.taxBookingAccounts[a.Number] = struct{}{}
				}
			}
		}
	}
}

func (p *Payment) isIncludingTax() bool {
	return p.Amount.TaxHandling == "incl"
}

// getValue returns the value of that Receipt in Cents.
func (p *Payment) getValue() jesva.Cents {
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

	p.Amount.value = jesva.Cents(ival)
	return p.Amount.value
}

// getNetAmount returns the net amount of this Payment, i.e. without taxes.
func (p *Payment) getNetAmount(perc int) jesva.Cents {
	val := p.getValue()

	if !p.isIncludingTax() || perc == 0 {
		return val
	}

	return val.NetAmount(perc)
}

// getTax returns the taxes of this Payment.
func (p *Payment) getTax(perc int) jesva.Cents {
	if perc == 0 {
		return jesva.Cents(0)
	}

	val := p.getValue()

	if p.isIncludingTax() {
		amt := p.getNetAmount(perc)
		return val - amt
	}

	return val.Percentage(perc)
}

func (e *Eur) payments(period jesva.Period) iter.Seq[*Payment] {
	return func(yield func(*Payment) bool) {
		for _, r := range e.Receipts {
			if r.Paid && period.Includes(r.Date) {
				for _, p := range r.Payments {
					// tax bookings are not relevant to taxes itself, so we ignore them
					_, isTaxBookingAccount := e.taxBookingAccounts[p.Account]
					if !isTaxBookingAccount {
						if !yield(p) {
							return
						}
					}
				}
			}
		}
	}
}

type VatDataEntry struct {
	Tax       jesva.Cents
	NetAmount jesva.Cents
	Percent   int
}

func (v VatDataEntry) Empty() bool {
	return v.Tax == 0 && v.NetAmount == 0
}

type VatData map[TaxAccount]VatDataEntry

// VatData returns amount and vat amount for each account in the given period.
func (e *Eur) VatData(period jesva.Period) VatData {
	vatData := make(VatData, len(e.accountInfo))

	type paymentWithAccount struct {
		*Payment
		acc TaxAccount
	}

	perAccount := func(p paymentWithAccount) {
		acc := e.accountInfo[p.acc]
		taxDiff := p.getTax(acc.Percent)
		amountDiff := p.getNetAmount(acc.Percent)

		jesva.Debug("Kto %02d/%02d (#%d):\t%s / %s", p.acc, p.Account, p.receipt.Number,
			amountDiff.Format("%3d.%02d EUR"),
			taxDiff.Format("%3d.%02d EUR"))

		vd := vatData[p.acc]
		vd.Tax += taxDiff
		vd.NetAmount += amountDiff
		vd.Percent = acc.Percent
		vatData[p.acc] = vd
	}

	// to help debugging, we sort the payments by account number and receipt number
	payments := make([]paymentWithAccount, 0, 100)

	for p := range e.payments(period) {
		if p.Incoming != 0 {
			payments = append(payments, paymentWithAccount{p, p.Incoming})
		}
		if p.Outgoing != 0 {
			payments = append(payments, paymentWithAccount{p, p.Outgoing})
		}
	}

	slices.SortFunc(payments, func(a, b paymentWithAccount) int {
		return cmp.Or(cmp.Compare(a.acc, b.acc), cmp.Compare(a.receipt.Number, b.receipt.Number))
	})

	for _, p := range payments {
		perAccount(p)
	}

	return vatData
}

func (e *Eur) Validate(knownAccounts []TaxAccount) {
	if e.Start.Year() != e.End.Year() {
		log.Fatalf("JES spans multiple years. This is not supported.")
	}

	accounts := make(map[TaxAccount]struct{})
	for _, acc := range knownAccounts {
		accounts[acc] = struct{}{}
	}

	// Check for unsupported tax accounts
	checkFn := func(acc TaxAccount) {
		if acc == 0 { // account not given
			return
		}

		if _, ok := accounts[acc]; !ok {
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

func Read(jesFile string) *Eur {
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
