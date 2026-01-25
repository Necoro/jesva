package main

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
)

// Mapping of JES account types to Elster-Kennzahlen.
// `typ` specifies whether we sum gross amounts or taxes.
type Mapping struct {
	kz      int
	account int
	typ     SumType
}

const (
	// NA is used for accounts that are not mapped.
	NA = 0
	// MinExpenseAccount is the lowest tax account number that is considered an expense.
	// All accounts below this number are considered income.
	MinExpenseAccount = 500
)

var mappings = []Mapping{
	// Steuerpflichtige Umsätze 19%
	{81, 500, Amount},
	// Steuerpflichtige Umsätze 7%
	{83, 510, Amount},
	// Steuerpflichtige Umsätze 0%
	// This is not reproduced in JES, as there is a difference between taxed with 0% and taxfree.
	// Account 520 is used for taxfree, and is therefore not applicable here.
	// USt 0% (Steuerfrei) --> Ignore
	{NA, 520, Ignore},
	// VSt 19%
	{66, 100, Tax},
	// VSt 7%
	{66, 110, Tax},
	// Vst 0% --> Ignore
	{NA, 120, Ignore},
	// §13b UStG USt
	{46, 600, Amount},
	{47, 600, Tax},
	// §13b UStG VSt
	{67, 200, Tax},
	// Leistungsempfänger schuldet USt
	{60, 610, Amount},
	// Innergemeinschaftlicher Erwerb
	{89, 650, Amount},
	{93, 655, Amount},
	{61, 250, Tax},
	{61, 255, Tax},
	// TODO: Einfuhrumsatzsteuer
}

const (
	// Sondervorauszahlung
	KzSvz = 39
)

// as defined by Elster
const header = `<?xml version="1.0" encoding="ISO-8859-15" standalone="no"?>` + "\n"

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
	Jahr         int    `xml:"Jahr"`
	Zeitraum     string `xml:"Zeitraum"`
	Steuernummer string `xml:"Steuernummer"`
	WIdNr        string `xml:"WIdNr,omitempty"`
	Kennzahlen   Kennzahlen
}

// Kennzahl is the content of one field on the UStVA form.
type Kennzahl struct {
	withFraction bool
	amount       Cents
}

// Kennzahlen represents all filled fields on the UStVA form.
// It maps the field number to its content.
type Kennzahlen map[int]Kennzahl

func (k Kennzahl) amountString() string {
	euro, cents := k.amount.AsEuro()

	if k.withFraction {
		return fmt.Sprintf("%d.%02d", euro, cents)
	} else {
		return fmt.Sprintf("%d", euro)
	}
}

func (k Kennzahl) relevantAmount() Cents {
	if k.withFraction {
		return k.amount
	} else {
		return k.amount.FullEuros()
	}
}

// kennzahlenFromJES processes the JES receipts and calculates the Kennzahlen fields of the UStVA form.
func kennzahlenFromJES(jesData *Eur, period Period) Kennzahlen {
	kennzahlen := make(Kennzahlen)

	for _, m := range mappings {
		if m.typ == Ignore {
			continue
		}

		val := jesData.ReceiptSum(m.account, m.typ, period)
		if val != 0 {
			kz, ok := kennzahlen[m.kz]
			if ok {
				kz.amount += val
			} else {
				kz = Kennzahl{
					withFraction: m.typ == Tax,
					amount:       val,
				}
			}
			debug("\t\t=> Kz %02d:\t%s\t(= %s)", m.kz, val, kz.amountString())
			kennzahlen[m.kz] = kz
		}
	}

	return kennzahlen
}

// fillUStVA generates the content for the UStVA fields.
func fillUStVA(conf *Config, jesData *Eur, period Period, svz Cents) UStVA {
	ustva := UStVA{
		Jahr:         jesData.Year(),
		Zeitraum:     period.String(),
		Steuernummer: conf.UStNr,
		WIdNr:        conf.WIdNr,
		Kennzahlen:   kennzahlenFromJES(jesData, period),
	}

	if svz != 0 {
		kz := Kennzahl{withFraction: true, amount: svz}
		ustva.Kennzahlen[KzSvz] = kz
		debug("\t\t=> Kz %02d:\t%s\t(= %s)", KzSvz, svz, kz.amountString())
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

func fillDatenlieferant(conf *Config, jesData *Eur) Datenlieferant {
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

func fillUnternehmer(conf *Config, jesData *Eur) Unternehmer {
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
func WriteVatFile(w io.Writer, conf *Config, jesData *Eur, period Period, svz Cents) {
	// ISO-8859-15 is requested
	isoEncoder := charmap.ISO8859_15.NewEncoder()
	w = isoEncoder.Writer(w)

	// fill data
	a := anmeldungForYear(jesData.Year())
	a.Datenlieferant = fillDatenlieferant(conf, jesData)
	a.Unternehmer = fillUnternehmer(conf, jesData)
	a.UStVA = fillUStVA(conf, jesData, period, svz)

	// write the header
	if _, err := io.WriteString(w, header); err != nil {
		log.Fatalf("Writing XML: %v", err)
	}

	// encode to XML
	xmlEncoder := xml.NewEncoder(w)
	xmlEncoder.Indent("", "    ") // indentation is nice for debugging
	defer xmlEncoder.Close()

	if err := xmlEncoder.Encode(a); err != nil {
		log.Fatalf("Error encoding XML: %v", err)
	}

	sum := CalculateVatSum(jesData, a.UStVA.Kennzahlen)
	fmt.Fprintf(os.Stderr, "*** Expected Tax Sum: %s", sum)
	if svz != 0 {
		fmt.Fprintf(os.Stderr, " (observing SVZ: %s)", sum-svz)
	}
	fmt.Fprintf(os.Stderr, " ***\n")
}

func cleanedMappings() []Mapping {
	m := slices.Clone(mappings)

	taxMappings := make(map[int]struct{})
	for _, m := range m {
		if m.typ == Tax {
			taxMappings[m.account] = struct{}{}
		}
	}

	return slices.DeleteFunc(m, func(m Mapping) bool {
		if m.typ == Ignore {
			return true
		}

		if _, ok := taxMappings[m.account]; ok && m.typ == Amount {
			return true
		}

		return false
	})
}

func CalculateVatSum(jesData *Eur, kennzahlen Kennzahlen) Cents {
	var sum Cents

	seen := make(map[int]struct{})

	for _, m := range cleanedMappings() {
		if _, ok := seen[m.kz]; ok {
			continue
		}
		seen[m.kz] = struct{}{}

		if kz, ok := kennzahlen[m.kz]; ok {
			var amt Cents
			if m.typ == Tax {
				amt = kz.relevantAmount()
			} else {
				perc := jesData.accountInfo[m.account].Percent
				amt = kz.relevantAmount().Percentage(perc)
			}

			if m.account < MinExpenseAccount {
				amt = -amt
			}

			debug("* %d => %s", m.kz, amt)

			sum += amt
		}
	}
	return sum
}

// BuildVatFile prints the UStVA XML to Stdout.
func BuildVatFile(conf *Config, jesData *Eur, period Period, svz Cents) {
	WriteVatFile(os.Stdout, conf, jesData, period, svz)
}
