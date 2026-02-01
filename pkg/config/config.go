package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
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

// Read loads the configuration from the location specified in `configName`
func Read() *Config {
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
