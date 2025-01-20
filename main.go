package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
)

const configName = "config.json"

// Config hold UStVA specific configuration that is not part of JES.
// More details can be found in the config.example.json
type Config struct {
	UStNr     string `json:"ustnr"`
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

type Period interface {
	includes(Date) bool
	String() string
}

type Month uint8

func (m Month) includes(d Date) bool {
	return d.Month == int(m)
}

func (m Month) String() string {
	return fmt.Sprintf("%02d", m)
}

type Quarter uint8

const (
	Q1 Quarter = iota + 1
	Q2
	Q3
	Q4
)

func (q Quarter) includes(d Date) bool {
	end := int(q) * 3
	return d.Month <= end && d.Month > end-3
}

func (q Quarter) String() string {
	// 4x = Qx
	return fmt.Sprintf("4%d", q)
}

// readConfig loads the configuration from the location specified in `configName`
func readConfig() *Config {
	f, err := os.Open(configName)
	if err != nil {
		log.Fatalf("Reading config at '%s': %v", configName, err)
	}

	config := new(Config)
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()

	if err = d.Decode(config); err != nil {
		log.Fatalf("Parsing config at '%s': %v", configName, err)
	}

	return config
}

func main() {
	log.SetFlags(0) // no prefix for logging

	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <jes.file> <period>\n\n<period> is either 1-12 for the months, or Q1-Q4 for the quarters", os.Args[0])
	}

	jesFile := os.Args[1]
	periodStr := os.Args[2]

	var period Period
	if periodStr[0] == 'q' || periodStr[0] == 'Q' {
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
	} else {
		month, err := strconv.Atoi(periodStr)
		if err != nil {
			log.Fatalf("Invalid month: %s (%v)", periodStr, err)
		}

		period = Month(month)
	}

	conf := readConfig()
	jes := readJesFile(jesFile)
	jes.Validate()

	BuildVatFile(conf, jes, period)
}
