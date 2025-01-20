package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

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
	Bezeichnung string `xml:"Bezeichnung,omitifempty"`
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

type UStVA struct {
	Jahr         string `xml:"Jahr"`
	Zeitraum     string `xml:"Zeitraum"`
	Steuernummer string `xml:"Steuernummer"`
}

func buildVatFile(conf *Config, jesData *Eur, month string) {
	writeVatFile(os.Stdout, conf, jesData, month)
}

func anmeldungForYear(year string) *Anmeldung {
	name := xml.Name{
		Local: "Anmeldungssteuern",
		Space: "http://finkonsens.de/elster/elsteranmeldung/ustva/v" + year,
	}

	now := time.Now()

	anmeldung := Anmeldung{
		XMLName: name,
		Version: year,
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

func fillUStVA(conf *Config, jesData *Eur, month string) UStVA {
	ustva := UStVA{
		Jahr:         jesData.Year(),
		Zeitraum:     month,
		Steuernummer: conf.UStNr,
	}

	return ustva
}

func writeVatFile(w io.Writer, conf *Config, jesData *Eur, month string) {

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
