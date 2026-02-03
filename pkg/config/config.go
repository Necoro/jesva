package config

import (
	"encoding/json"
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

// ReadFrom loads the configuration from the path passed in
func ReadFrom(path string) *Config {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Reading config at '%s': %v", path, err)
	}

	defer f.Close()

	config := new(Config)
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()

	if err = d.Decode(config); err != nil {
		log.Fatalf("Parsing config at '%s': %v", path, err)
	}

	return config
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

// Read loads the configuration from the location specified in `configName`
func Read() *Config {
	if exists(configName) {
		return ReadFrom(configName)
	}
	if exists(configAltName) {
		return ReadFrom(configAltName)
	}
	log.Fatalf("No config file ('%s' or '%s') found.", configName, configAltName)
	return nil
}
