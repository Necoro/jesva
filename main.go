package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"strings"
)

const (
	configName    = "config.json"
	configAltName = "jesva.json"
)

// Config hold UStVA specific configuration that is not part of JES.
// More details can be found in the config.example.json
type Config struct {
	UStNr     string `json:"ustnr"`
	WIdNr     string `json:"widnr"`
	Name      string `json:"name"`
	FirstName string `json:"firstName"`
	Address   struct {
		Street       string `json:"street"`
		Number       string `json:"number"`
		NumberSuffix string `json:"suffix"`
		Plz          string `json:"plz"`
		City         string `json:"city"`
	}
	Contact struct {
		Telephone string `json:"tel"`
		Mail      string `json:"mail"`
	}
}

// readConfig loads the configuration from the location specified in `configName`
func readConfig() *Config {
	name := configName

	f, err := os.Open(name)
	if errors.Is(err, fs.ErrNotExist) {
		name = configAltName
		f, err = os.Open(name)

		if errors.Is(err, fs.ErrNotExist) {
			log.Fatalf("No config file ('%s' or '%s') found.", configName, configAltName)
		}
	}

	if err != nil {
		log.Fatalf("Reading config at '%s': %v", name, err)
	}

	config := new(Config)
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()

	if err = d.Decode(config); err != nil {
		log.Fatalf("Parsing config at '%s': %v", name, err)
	}

	return config
}

var _debug = false

func debug(format string, args ...any) {
	if _debug {
		log.Printf(format, args...)
	}
}

func main() {
	log.SetFlags(0) // no prefix for logging
	log.SetOutput(os.Stderr)

	args := os.Args[1:]

	var svz Cents
cmdparsing:
	for len(args) >= 1 && len(args[0]) > 0 && args[0][0] == '-' {
		switch args[0] {
		case "-d":
			_debug = true
			args = args[1:]
		case "-svz":
			if len(args) < 2 || len(args[1]) == 0 {
				log.Fatalf("Missing amount for -svz option.")
			}
			svzStr := args[1]
			var err error
			if svz, err = ParseCents(svzStr); err != nil {
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

	var period Period
	if periodStr[0] == 'q' || periodStr[0] == 'Q' { // Quarter
		switch periodStr[1] {
		case '1':
			period = Q1
		case '2':
			period = Q2
		case '3':
			period = Q3
		case '4':
			period = Q4
		default:
			log.Fatalf("Unknown quarter '%s'", periodStr)
		}
	} else if startStr, endStr, found := strings.Cut(periodStr, "-"); found { // Month range
		start := parseMonth(startStr)
		end := parseMonth(endStr)

		if end <= start {
			log.Fatalf("End month must be larger than starting month.")
		}

		period = Months{start, end}
	} else if len(periodStr) <= 2 { // single month
		period = parseMonth(periodStr)
	} else if len(periodStr) == 4 { // year
		period = parseYear(periodStr)
	} else {
		log.Fatalf("Unknown period '%s'", periodStr)
	}

	conf := readConfig()
	jes := readJesFile(jesFile)
	jes.Validate()

	if _, ok := period.(Year); ok {
		// UStE
		xmls := args[2:]
		OutputUStE(jes, period, xmls)
	} else {
		// UStVA
		BuildVatFile(conf, jes, period, svz)
	}
}
