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

type Mapping struct {
	kz      int
	account int
	amount  bool
}

var kzToAccounts = []Mapping{
	{81, 500, true},
	{83, 510, true},
}

// as defined by Elster
const header = `<?xml version="1.0" encoding="ISO-8859-15" standalone="no"?>` + "\n"

type Anmeldung struct {
	XMLName        xml.Name
	Version        string         `xml:"version,attr"`
	Date           string         `xml:"Erstellungsdatum"`
	Datenlieferant Datenlieferant `xml:"DatenLieferant"`
	Unternehmer    Unternehmer    `xml:"Steuerfall>Unternehmer"`
	UStVA          UStVA          `xml:"Steuerfall>Umsatzsteuervoranmeldung"`
}

type Datenlieferant struct {
	Name    string `xml:"Name"`
	Strasse string `xml:"Strasse"`
	PLZ     string `xml:"PLZ"`
	Ort     string `xml:"Ort"`
	Telefon string `xml:"Telefon,omitempty"`
	Email   string `xml:"Email,omitempty"`
}

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

type Kennzahl struct {
	account *Account
	amount  Cents
}

func (k Kennzahl) isRounded() bool {
	return k.account.NeedsRounding()
}

func (k Kennzahl) amountString() string {
	integer := k.amount / 100

	if k.isRounded() {
		return fmt.Sprintf("%d", integer)
	} else {
		frac := k.amount % 100
		return fmt.Sprintf("%d.%02d", integer, frac)
	}
}

type Kennzahlen map[int]Kennzahl

func (k Kennzahlen) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	for key, val := range k {
		if val.account == nil {
			log.Fatalf("No account info for Kennzahl %d.", key)
		}

		name := xml.Name{Local: fmt.Sprintf("Kz%02d", key)}
		se := xml.StartElement{Name: name}

		amount := val.amountString()

		e.EncodeToken(se)
		e.EncodeToken(xml.CharData([]byte(amount)))
		e.EncodeToken(se.End())
	}
	return nil
}

type UStVA struct {
	Jahr         int    `xml:"Jahr"`
	Zeitraum     string `xml:"Zeitraum"`
	Steuernummer string `xml:"Steuernummer"`
	Kennzahlen   Kennzahlen
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

func fillUStVA(conf *Config, jesData *Eur, month int) UStVA {
	ustva := UStVA{
		Jahr:         jesData.Year(),
		Zeitraum:     fmt.Sprintf("%02d", month),
		Steuernummer: conf.UStNr,
		Kennzahlen:   make(map[int]Kennzahl),
	}

	for _, m := range kzToAccounts {
		val, acc := jesData.ReceiptValue(m.account, m.amount, month)
		if val != 0 {
			ustva.Kennzahlen[m.kz] = Kennzahl{acc, val}
		}
	}

	return ustva
}

func writeVatFile(w io.Writer, conf *Config, jesData *Eur, month int) {
	// ISO-8859-15 is requested
	isoEncoder := charmap.ISO8859_15.NewEncoder()
	w = isoEncoder.Writer(w)

	// start with header
	if _, err := io.WriteString(w, header); err != nil {
		log.Fatalf("Writing XML: %v", err)
	}

	xmlEncoder := xml.NewEncoder(w)
	xmlEncoder.Indent("", "    ")
	defer xmlEncoder.Close()

	a := anmeldungForYear(jesData.Year())
	a.Datenlieferant = fillDatenlieferant(conf, jesData)
	a.Unternehmer = fillUnternehmer(conf, jesData)
	a.UStVA = fillUStVA(conf, jesData, month)

	if err := xmlEncoder.Encode(a); err != nil {
		log.Fatalf("Error encoding XML: %v", err)
	}
}

func buildVatFile(conf *Config, jesData *Eur, month int) {
	writeVatFile(os.Stdout, conf, jesData, month)
}
