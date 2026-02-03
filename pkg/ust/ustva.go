package ust

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"

	"github.com/Necoro/jesva/pkg/config"
	"github.com/Necoro/jesva/pkg/jes"
	"github.com/Necoro/jesva/pkg/jesva"
)

// as defined by Elster
const (
	header     = `<?xml version="1.0" encoding="ISO-8859-15" standalone="no"?>` + "\n"
	headerUTF8 = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>` + "\n"
)

// Anmeldung is the final XML structure requested by Elster.
type Anmeldung struct {
	XMLName        xml.Name
	Version        string         `xml:"version,attr"`
	Date           string         `xml:"Erstellungsdatum"`
	Datenlieferant Datenlieferant `xml:"DatenLieferant"`
	Unternehmer    Unternehmer    `xml:"Steuerfall>Unternehmer"`
	UStVA          UStVA          `xml:"Steuerfall>Umsatzsteuervoranmeldung"`
}

// Datenlieferant holds data about who processed the data.
// We don't make any distinction and also fill it with the data from the enterprise.
// Unsure how it actually matters.
type Datenlieferant struct {
	Name    string `xml:"Name"`
	Strasse string `xml:"Strasse"`
	PLZ     string `xml:"PLZ"`
	Ort     string `xml:"Ort"`
	Telefon string `xml:"Telefon,omitempty"`
	Email   string `xml:"Email,omitempty"`
}

// Unternehmer holds the businesses general data.
// Most of it is *not* part of JES and needs to provided additionally.
type Unternehmer struct {
	Bezeichnung string `xml:"Bezeichnung,omitempty"`
	Name        string `xml:"Name"`
	Vorname     string `xml:"Vorname"`
	Strasse     string `xml:"Str"`
	Hausnummer  string `xml:"Hausnummer"`
	HNrZusatz   string `xml:"HNrZusatz,omitempty"`
	Ort         string `xml:"Ort"`
	PLZ         string `xml:"PLZ"`
	Telefon     string `xml:"Telefon,omitempty"`
	Email       string `xml:"Email,omitempty"`
}

// UStVA holds the actual tax relevant content.
type UStVA struct {
	Jahr         int        `xml:"Jahr"`
	Zeitraum     string     `xml:"Zeitraum"`
	Steuernummer string     `xml:"Steuernummer"`
	WIdNr        string     `xml:"WIdNr,omitempty"`
	Kennzahlen   Kennzahlen `xml:",any"`
}

// Kennzahl is the content of one field on the UStVA form.
type Kennzahl struct {
	withFraction bool
	amount       jesva.Cents
	account      jes.TaxAccount
	percent      int
	typ          SumType
}

// Kennzahlen represents all filled fields on the UStVA form.
// It maps the field number to its content.
type Kennzahlen map[int]*Kennzahl

func (k *Kennzahl) amountString() string {
	euro, cents := k.amount.AsEuro()

	if k.withFraction {
		return fmt.Sprintf("%d.%02d", euro, cents)
	} else {
		return fmt.Sprintf("%d", euro)
	}
}

func (k *Kennzahl) relevantAmount() jesva.Cents {
	if k.withFraction {
		return k.amount
	} else {
		return k.amount.FullEuros()
	}
}

func (k *Kennzahl) taxAmount() jesva.Cents {
	switch k.typ {
	case AmountOnly, Ignore:
		return 0
	case Tax:
		return k.relevantAmount()
	case Amount:
		return k.relevantAmount().Percentage(k.percent)
	}
	return 0
}

func (k Kennzahlen) Merge(id int, kz Kennzahl) {
	other, ok := k[id]
	if !ok {
		k[id] = &kz
		return
	}

	// Assertions of consistency
	if kz.typ != Tax && kz.percent != other.percent {
		log.Fatalf("Inconsistent tax rate for Kz %d: %d vs %d", id, kz.percent, other.percent)
	}
	if kz.typ != other.typ {
		log.Fatalf("Inconsistent mapping for Kz %d: %d vs %d", id, kz.typ, other.typ)
	}
	if kz.account.IsExpense() != other.account.IsExpense() {
		log.Fatalf("Expense account mixed with income account for Kz %d", id)
	}

	k[id].amount += kz.amount
}

func (k Kennzahlen) TaxSum() jesva.Cents {
	var sum jesva.Cents

	sortedKeys := slices.Sorted(maps.Keys(k))

	for _, id := range sortedKeys {
		kz := k[id]
		amt := kz.taxAmount()
		jesva.Debug("* %d => %s", id, amt)

		if kz.account.IsExpense() {
			amt = -amt
		}

		sum += amt
	}
	return sum
}

// kennzahlenFromVatData processes the JES receipts and calculates the Kennzahlen fields of the UStVA form.
func kennzahlenFromVatData(vatData jes.VatData) Kennzahlen {
	kennzahlen := make(Kennzahlen)

	for _, m := range mappings {
		if m.typ == Ignore {
			continue
		}

		vat, ok := vatData[m.account]
		if !ok {
			continue
		}

		if !vat.Empty() {
			var val jesva.Cents
			switch m.typ {
			case Amount, AmountOnly:
				val = vat.NetAmount
			case Tax:
				val = vat.Tax
			default:
				log.Fatalf("Unknown sum type %d", m.typ)
			}

			kz := Kennzahl{
				withFraction: m.typ == Tax,
				amount:       val,
				typ:          m.typ,
				account:      m.account,
				percent:      vat.Percent,
			}
			kennzahlen.Merge(m.kz, kz)

			jesva.Debug("\t=> Kz %02d (Kto %d, %s):\t%s\t(= %s)", m.kz, m.account, m.typ, val, kz.amountString())
		}
	}

	return kennzahlen
}

// fillUStVA generates the content for the UStVA fields.
func fillUStVA(conf *config.Config, jesData *jes.Eur, period jesva.Period, svz jesva.Cents) UStVA {
	vatData := jesData.VatData(period)
	ustva := UStVA{
		Jahr:         jesData.Year(),
		Zeitraum:     period.String(),
		Steuernummer: conf.UStNr,
		WIdNr:        conf.WIdNr,
		Kennzahlen:   kennzahlenFromVatData(vatData),
	}

	if svz != 0 {
		kz := Kennzahl{withFraction: true, amount: svz, typ: Tax, account: 0}
		ustva.Kennzahlen.Merge(KzSvz, kz)
		jesva.Debug("\t=> Kz %02d (SVZ):\t\t\t%s\t(= %s)", KzSvz, svz, kz.amountString())
	}

	return ustva
}

// MarshalXML converts the Kennzahlen map into the <KzXY> structure.
func (k Kennzahlen) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	sortedKeys := slices.Sorted(maps.Keys(k))

	for _, key := range sortedKeys {
		val := k[key]
		name := xml.Name{Local: fmt.Sprintf("Kz%02d", key)}
		se := xml.StartElement{Name: name}

		amount := val.amountString()

		e.EncodeToken(se)
		e.EncodeToken(xml.CharData(amount))
		e.EncodeToken(se.End())
	}
	return nil
}

func anmeldungForYear(year int) *Anmeldung {
	yearStr := strconv.Itoa(year)

	name := xml.Name{
		Local: "Anmeldungssteuern",
		Space: "http://finkonsens.de/elster/elsteranmeldung/ustva/v" + yearStr,
	}

	now := time.Now()

	anmeldung := Anmeldung{
		XMLName: name,
		Version: yearStr,
		Date:    now.Format("20060102"),
	}

	return &anmeldung
}

func fillDatenlieferant(conf *config.Config, jesData *jes.Eur) Datenlieferant {
	name := jesData.Name

	if conf.Name != "" && conf.FirstName != "" {
		name = conf.FirstName + " " + conf.Name
	}

	return Datenlieferant{
		Name:    name,
		Strasse: fmt.Sprintf("%s %s%s", conf.Address.Street, conf.Address.Number, conf.Address.NumberSuffix),
		PLZ:     conf.Address.Plz,
		Ort:     conf.Address.City,
		Telefon: conf.Contact.Telephone,
		Email:   conf.Contact.Mail,
	}
}

func fillUnternehmer(conf *config.Config, jesData *jes.Eur) Unternehmer {
	firstName, lastName, _ := strings.Cut(jesData.Name, " ")

	if conf.FirstName != "" {
		firstName = conf.FirstName
	}
	if conf.Name != "" {
		lastName = conf.Name
	}

	return Unternehmer{
		Bezeichnung: jesData.Company,
		Name:        lastName,
		Vorname:     firstName,
		Strasse:     conf.Address.Street,
		Hausnummer:  conf.Address.Number,
		HNrZusatz:   conf.Address.NumberSuffix,
		PLZ:         conf.Address.Plz,
		Ort:         conf.Address.City,
		Telefon:     conf.Contact.Telephone,
		Email:       conf.Contact.Mail,
	}
}

// WriteVatFile writes the UStVA XML to the given Writer.
func WriteVatFile(w io.Writer, utf8 bool, conf *config.Config, jesData *jes.Eur, period jesva.Period, svz jesva.Cents) {
	if !utf8 {
		// ISO-8859-15 is requested
		isoEncoder := charmap.ISO8859_15.NewEncoder()
		w = isoEncoder.Writer(w)
	}

	// fill data
	a := anmeldungForYear(jesData.Year())
	a.Datenlieferant = fillDatenlieferant(conf, jesData)
	a.Unternehmer = fillUnternehmer(conf, jesData)
	a.UStVA = fillUStVA(conf, jesData, period, svz)

	// write the header
	if utf8 {
		if _, err := io.WriteString(w, headerUTF8); err != nil {
			log.Fatalf("Writing XML: %v", err)
		}
	} else {
		if _, err := io.WriteString(w, header); err != nil {
			log.Fatalf("Writing XML: %v", err)
		}
	}

	// encode to XML
	xmlEncoder := xml.NewEncoder(w)
	xmlEncoder.Indent("", "    ") // indentation is nice for debugging
	defer xmlEncoder.Close()

	if err := xmlEncoder.Encode(a); err != nil {
		log.Fatalf("Error encoding XML: %v", err)
	}

	taxSum := a.UStVA.Kennzahlen.TaxSum()
	fmt.Fprintf(os.Stderr, "*** Expected Tax Sum: %s ***\n", taxSum)
}

// BuildVatFile prints the UStVA XML to Stdout.
func BuildVatFile(conf *config.Config, jesData *jes.Eur, period jesva.Period, svz jesva.Cents) {
	WriteVatFile(os.Stdout, false, conf, jesData, period, svz)
}
