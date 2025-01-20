package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
)

const configName = "config.json"

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
	log.SetFlags(0)
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <jes.file> <month>", os.Args[0])
	}

	jesFile := os.Args[1]
	monthStr := os.Args[2]

	month, err := strconv.Atoi(monthStr)
	if err != nil {
		log.Fatalf("Invalid month: %s (%v)", monthStr, err)
	}

	conf := readConfig()

	jes := readJesFile(jesFile)

	//log.Printf("%+v", jes)

	buildVatFile(conf, jes, month)
}
