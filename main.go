package main

import (
	"log"
	"os"
	"strings"

	"github.com/Necoro/jesva/pkg/config"
	"github.com/Necoro/jesva/pkg/jes"
	"github.com/Necoro/jesva/pkg/jesva"
	"github.com/Necoro/jesva/pkg/ust"
)

func main() {
	log.SetFlags(0) // no prefix for logging
	log.SetOutput(os.Stderr)

	args := os.Args[1:]

	var svz jesva.Cents
cmdparsing:
	for len(args) >= 1 && len(args[0]) > 0 && args[0][0] == '-' {
		switch args[0] {
		case "-d":
			jesva.EnableDebug()
			args = args[1:]
		case "-svz":
			if len(args) < 2 || len(args[1]) == 0 {
				log.Fatalf("Missing amount for -svz option.")
			}
			svzStr := args[1]
			var err error
			if svz, err = jesva.ParseCents(svzStr); err != nil {
				log.Fatalf("Parsing -svz option: %v", err)
			}

			args = args[2:]
		default:
			break cmdparsing
		}
	}

	if len(args) < 2 {
		log.Fatalf(`Usage: %s [options] <jes.file> <period>

<period> is either:
	* 1,...,12 for a month
	* Q1,...,Q4 for a quarter

Possible options:
	-d: Enable debug output
	-svz amount: Take into account a Sondervorauszahlung.

Additionally, there exists the year-end mode:
> %[1]s [options] <jes.file> <year> <xml-file 1, xml-file 2, ..., xml-file n>
`, os.Args[0])
	}

	jesFile := args[0]
	periodStr := args[1]

	var period jesva.Period
	if periodStr[0] == 'q' || periodStr[0] == 'Q' { // Quarter
		period = jesva.ParseQuarter(periodStr)
	} else if startStr, endStr, found := strings.Cut(periodStr, "-"); found { // Month range
		period = jesva.ParseMonths(startStr, endStr)
	} else if len(periodStr) <= 2 { // single month
		period = jesva.ParseMonth(periodStr)
	} else if len(periodStr) == 4 { // year
		period = jesva.ParseYear(periodStr)
	} else {
		log.Fatalf("Unknown period '%s'", periodStr)
	}

	conf := config.Read()
	eur := jes.Read(jesFile)
	eur.Validate(ust.KnownAccounts())

	if _, ok := period.(jesva.Year); ok {
		// UStE
		xmls := args[2:]
		ust.OutputUStE(eur, period, xmls)
	} else {
		// UStVA
		ust.BuildVatFile(conf, eur, period, svz)
	}
}
