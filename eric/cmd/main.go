package main

import "C"
import (
	"log"
	"os"
	"strings"

	"github.com/Necoro/jesva/eric"
	"github.com/Necoro/jesva/pkg/config"
	"github.com/Necoro/jesva/pkg/jes"
	"github.com/Necoro/jesva/pkg/jesva"
	"github.com/Necoro/jesva/pkg/ust"
)

func run(c *config.Config, eur *jes.Eur, p jesva.Period) {
	e := new(eric.Eric)
	e.Init()
	defer e.Close()

	if err := e.CheckSteuerNr(c.UStNr); err != nil {
		log.Fatal(err)
	}

	if err := e.CheckWID(c.WIdNr); err != nil {
		log.Fatal(err)
	}

	builder := strings.Builder{}

	ust.WriteVatFile(&builder, true, c, eur, p, 0)

	xml := builder.String()
	_ = e.CheckXML(xml, eric.GetDatenart(eric.UStVA, 2026))
}

func main() {
	var p jesva.Period
	if pStr, _ := os.LookupEnv("PERIOD"); pStr != "" {
		var err error
		if p, err = jesva.Parse(pStr); err != nil {
			log.Fatalf("Parsing period: %v", err)
		}
	} else {
		log.Fatalf("PERIOD environment variable not set")
	}

	var c *config.Config
	if cfgPath, _ := os.LookupEnv("PROFILE"); cfgPath != "" {
		c = config.ReadFrom(cfgPath)
	} else {
		c = config.Read()
	}

	var eur *jes.Eur
	if jesPath, _ := os.LookupEnv("JES_FILE"); jesPath != "" {
		eur = jes.Read(jesPath)
		eur.Validate(ust.KnownAccounts())
	} else {
		log.Fatalln("No JES_FILE set.")
	}

	run(c, eur, p)
}
