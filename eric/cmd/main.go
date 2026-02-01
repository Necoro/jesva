package main

import "C"
import (
	"fmt"
	"log"

	"github.com/Necoro/jesva/eric"
	"github.com/Necoro/jesva/pkg/config"
)

func test(e *eric.Eric) {
	d := eric.GetDatenart(eric.UStVA, 2026)
	fmt.Println("Datenart: " + d.String())
}

func main() {
	e := new(eric.Eric)
	e.Init()
	defer e.Close()

	test(e)

	e.SystemCheck()

	c := config.Read()
	if err := e.CheckSteuerNr(c.UStNr); err != nil {
		log.Fatal(err)
	}

	if err := e.CheckWID(c.WIdNr); err != nil {
		log.Fatal(err)
	}
}
