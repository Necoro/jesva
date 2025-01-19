package main

import (
	"compress/flate"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <jes.file> <month>", os.Args[0])
	}

	jesFile := os.Args[1]
	//month := os.Args[2]

	f, err := os.Open(jesFile)
	if err != nil {
		log.Fatalf("Opening file '%s': %v", jesFile, err)
	}
	defer f.Close()

	deflateReader := flate.NewReader(f)
	defer deflateReader.Close()
}
