package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

type UStELine uint16

func (l UStELine) String() string {
	if l > 1000 {
		return fmt.Sprintf("%02d/%02d", l/100, l%100)
	}
	return fmt.Sprintf("%02d   ", l) // three trailing spaces to accomodate for the optional fields
}

// UnmarshalXML implements xml.Unmarshaler.
// It expects XML elements of the form <Kz123>45.67</Kz123> and stores the value in the Kennzahlen map.
func (k *Kennzahlen) UnmarshalXML(d *xml.Decoder, elem xml.StartElement) error {
	if elem.Name.Local[0:2] != "Kz" {
		return fmt.Errorf("unexpected XML element: %s", elem.Name.Local)
	}

	kz, err := strconv.Atoi(elem.Name.Local[2:])
	if err != nil {
		return fmt.Errorf("invalid Kennzahl: %s", elem.Name.Local)
	}

	data := struct {
		Data string `xml:",chardata"`
	}{}
	if err = d.DecodeElement(&data, &elem); err != nil {
		return err
	}

	cents, err := ParseCents(data.Data)
	if err != nil {
		return err
	}

	if *k == nil {
		*k = make(Kennzahlen)
	}

	mIdx := slices.IndexFunc(mappings, func(m Mapping) bool { return m.kz == kz })
	if mIdx < 0 {
		k.Merge(kz, Kennzahl{typ: Ignore, amount: cents})
	} else {
		mapping := mappings[mIdx]

		kennzahl := Kennzahl{
			amount:       cents,
			withFraction: strings.Contains(data.Data, "."),
			typ:          mapping.typ,
			account:      mapping.account,
		}
		k.Merge(kz, kennzahl)
	}
	return nil
}

func readUStVAXml(xmlFile string) Kennzahlen {
	data, err := os.ReadFile(xmlFile)
	if err != nil {
		log.Fatalf("Could not read UStVA XML file '%s': %v", xmlFile, err)
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch charset {
		case "ISO-8859-15":
			return charmap.ISO8859_15.NewDecoder().Reader(input), nil
		default:
			return nil, fmt.Errorf("unexpected charset %s", charset)
		}
	}

	var anmeldung Anmeldung
	if err = dec.Decode(&anmeldung); err != nil {
		log.Fatalf("Error parsing UStVA XML file '%s': %v", xmlFile, err)
	}
	return anmeldung.UStVA.Kennzahlen
}

func OutputUStE(jes *Eur, period Period, xmls []string, svz Cents) {
	ustvas := make([]Kennzahlen, len(xmls))
	for i, xmlFile := range xmls {
		ustvas[i] = readUStVAXml(xmlFile)
	}

	combinedKz := make(Kennzahlen)
	for _, ustva := range ustvas {
		for kz, kzVal := range ustva {
			combinedKz.Merge(kz, *kzVal)
		}
	}

	for id, kz := range combinedKz {
		acc := combinedKz[id].account
		kz.percent = jes.accountInfo[acc].Percent
	}

	vatData := jes.VatData(period)
	fullYearKz := kennzahlenFromVatData(vatData)

	vzSum := combinedKz.TaxSum()
	fySum := fullYearKz.TaxSum()

	type lineKz struct {
		fy *Kennzahl
		vz *Kennzahl
	}

	byLine := make(map[UStELine]lineKz)
	for _, m := range mappings {
		if m.typ == Ignore {
			continue
		}

		vz := combinedKz[m.kz]
		fy := fullYearKz[m.kz]

		if vz == nil && fy == nil {
			continue
		}

		if fy == nil {
			// should not happen
			log.Fatalf("Missing full year value for Kennzahl %d", m.kz)
		}

		line, found := byLine[m.zeile]
		if found {
			line.fy.amount += fy.amount
			if vz != nil {
				line.vz.amount += vz.amount
			}
		} else {
			fyCopy := *fy

			if vz != nil {
				vzCopy := *vz
				line = lineKz{fy: &fyCopy, vz: &vzCopy}
			} else {
				line = lineKz{fy: &fyCopy, vz: nil}
			}
		}

		byLine[m.zeile] = line
	}

	lines := slices.Sorted(maps.Keys(byLine))

	for _, zeile := range lines {
		line := byLine[zeile]
		printLine(line.fy, line.vz, zeile)
	}

	sumKz := func(amt Cents) *Kennzahl {
		return &Kennzahl{typ: Tax, amount: amt, withFraction: true}
	}

	printLine(sumKz(fySum), sumKz(vzSum), 999)
}

func printLine(fullYear *Kennzahl, vz *Kennzahl, zeile UStELine) {
	delta := fullYear.taxAmount() - vz.taxAmount()

	if fullYear.typ == AmountOnly {
		delta = fullYear.relevantAmount() - vz.relevantAmount()
	}

	fmt.Printf(" %s\t=>\t%s", zeile, fullYear.relevantAmount().Format("%5d.%02d EUR"))

	if fullYear.typ == Amount {
		fmt.Printf("\t(%s", fullYear.taxAmount())
	}

	if delta != 0 {
		fmt.Printf("\tΔ %s", delta)
	}

	if fullYear.typ == Amount {
		fmt.Print(")")
	}

	fmt.Println()
}
