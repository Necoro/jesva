package main

import (
	"encoding/json"
	"log"
	"os"
)

const configName = "config.json"

type Config struct {
	UStNr string `json:"ustnr"`
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
	//month := os.Args[2]

	conf := readConfig()

	jes := readJesFile(jesFile)

	log.Printf("%+v", jes)

	buildVatFile(conf, jes)
}
