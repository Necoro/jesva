package main

import (
	"archive/zip"
	"encoding/xml"
	"log"
)

type Eur struct {
	XmlName  xml.Name  `xml:"eur"`
	Name     string    `xml:"general>name"`
	Address  string    `xml:"general>address"`
	Company  string    `xml:"general>company"`
	TaxID    string    `xml:"general>taxid"`
	Start    Date      `xml:"general>businessyearrange>daterange>start>date"`
	Receipts []Receipt `xml:"receipts>receipt"`
}

type Date struct {
	Year  string `xml:"year,attr"`
	Month string `xml:"month,attr"`
	Day   string `xml:"day,attr"`
}

type Receipt struct {
	Date     Date   `xml:"date"`
	Incoming string `xml:"payment>taxaccountincoming"`
	Outgoing string `xml:"payment>taxaccountoutgoing"`
	Amount   struct {
		TaxHandling string `xml:"tax,attr"`
		Value       string `xml:",chardata"`
	} `xml:"payment>amount"`
}

func readJesFile(jesFile string) *Eur {
	zipF, err := zip.OpenReader(jesFile)
	if err != nil {
		log.Fatalf("Opening file '%s': %v", jesFile, err)
	}
	defer zipF.Close()

	f, err := zipF.File[0].Open()
	if err != nil {
		log.Fatalf("Decompressing file '%s': %v", jesFile, err)
	}
	defer f.Close()

	eur := new(Eur)
	decoder := xml.NewDecoder(f)
	err = decoder.Decode(eur)
	if err != nil {
		log.Fatalf("Decoding '%s': %v", jesFile, err)
	}

	return eur
}
