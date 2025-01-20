package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
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

var mappings = []Mapping{
	// Steuerpflichtige Umsätze 19%
	{81, 500, Amount},
	// Steuerpflichtige Umsätze 7%
	{83, 510, Amount},
	// Steuerpflichtige Umsätze 0%
	{87, 520, Amount},
	// VSt 19%
	{66, 100, Tax},
	// VSt 7%
	{66, 110, Tax},
	// §13b UStG USt
	{46, 600, Amount},
	{47, 600, Tax},
	// §13b UStG VSt
	{67, 200, Tax},
	// Leistungsempfänger schuldet USt
	{61, 610, Amount},
	// TODO: Einfuhrumsatzsteuer
}

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
	integer := k.amount / 100

	if k.withFraction {
		frac := k.amount % 100
		return fmt.Sprintf("%d.%02d", integer, frac)
	} else {
		return fmt.Sprintf("%d", integer)
	}
}

// fillUStVA generates the content for the UStVA fields by processing the JES receipts.
func fillUStVA(conf *Config, jesData *Eur, period Period) UStVA {
	ustva := UStVA{
		Jahr:         jesData.Year(),
		Zeitraum:     period.String(),
		Steuernummer: conf.UStNr,
		Kennzahlen:   make(map[int]Kennzahl),
	}

	for _, m := range mappings {
		val := jesData.ReceiptSum(m.account, m.typ, period)
		if val != 0 {
			kz, ok := ustva.Kennzahlen[m.kz]
			if ok {
				kz.amount += val
			} else {
				kz = Kennzahl{
					withFraction: m.typ == Tax,
					amount:       val,
				}
			}
			ustva.Kennzahlen[m.kz] = kz
		}
	}

	return ustva
}

// MarshalXML converts the Kennzahlen map into the <KzXY> structure.
func (k Kennzahlen) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	for key, val := range k {
		name := xml.Name{Local: fmt.Sprintf("Kz%02d", key)}
		se := xml.StartElement{Name: name}

		amount := val.amountString()

		e.EncodeToken(se)
		e.EncodeToken(xml.CharData([]byte(amount)))
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
func WriteVatFile(w io.Writer, conf *Config, jesData *Eur, period Period) {
	// ISO-8859-15 is requested
	isoEncoder := charmap.ISO8859_15.NewEncoder()
	w = isoEncoder.Writer(w)

	// start with header
	if _, err := io.WriteString(w, header); err != nil {
		log.Fatalf("Writing XML: %v", err)
	}

	// fill data
	a := anmeldungForYear(jesData.Year())
	a.Datenlieferant = fillDatenlieferant(conf, jesData)
	a.Unternehmer = fillUnternehmer(conf, jesData)
	a.UStVA = fillUStVA(conf, jesData, period)

	// encode to XML
	xmlEncoder := xml.NewEncoder(w)
	xmlEncoder.Indent("", "    ") // indentation is nice for debugging
	defer xmlEncoder.Close()

	if err := xmlEncoder.Encode(a); err != nil {
		log.Fatalf("Error encoding XML: %v", err)
	}
}

// BuildVatFile prints the UStVA XML to Stdout.
func BuildVatFile(conf *Config, jesData *Eur, period Period) {
	WriteVatFile(os.Stdout, conf, jesData, period)
}
