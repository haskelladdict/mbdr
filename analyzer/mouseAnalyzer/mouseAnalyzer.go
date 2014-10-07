// mouseAnalyzer determines vesicle release events and latencies for our
// mouse NMJ 6 AZ model with two synaptic vesicles each according to the
// second sensor faciliation model (see Ma et al., J. Neurophys, 2014)
package main

import (
	"flag"
	"fmt"
	"github.com/haskelladdict/mbdr/libmbd"
	"log"
	"strconv"
	"strings"
)

var (
	numPulsesFlag int // number of AP pulses
	sytEnergyFlag int // energy of activated synaptotagmin toward vesicle fusion
	yEnergyFlag   int // energy of activated Y sites toward vesicle fusion
)

func init() {
	flag.IntVar(&numPulsesFlag, "n", 1, "number of AP pulses in the model")
	flag.IntVar(&sytEnergyFlag, "s", -1, "energy of active synaptotagmin sites")
	flag.IntVar(&yEnergyFlag, "y", -1, "energy of active y sites")
}

// usage prints a brief usage information to stdout
func usage() {
	fmt.Println("usage: mouseAnalyzer [options] <binary mcell files>")
	fmt.Println("\noptions:")
	flag.PrintDefaults()
}

// extractSeed attempts to extract the seed from the filename of the provided
// binary mcell data file.
// NOTE: the following filenaming convention is assumed *.<seedIDString>.bin.(gz|bz2)
func extractSeed(fileName string) (int, error) {
	items := strings.Split(fileName, ".")
	if len(items) <= 3 {
		return -1, fmt.Errorf("incorrectly formatted fileName. " +
			"Expected *.<seedIDString>.bin.(gz|bz2)")
	}

	for i := len(items) - 1; i >= 0; i-- {
		if items[i] == "bin" && i >= 1 {
			seed, err := strconv.Atoi(items[i-1])
			if err != nil {
				return -1, err
			}
			return seed, nil
		}
	}
	return -1, fmt.Errorf("Unable to extract seed id from filename ", fileName)
}

// main entry point
func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		usage()
		return
	}
	if sytEnergyFlag < 0 || yEnergyFlag < 0 {
		log.Fatal("Please provide a non-negative synaptotagmin and y site energy")
	}

	// loop over all provided data sets
	for _, fileName := range flag.Args() {
		seed, err := extractSeed(fileName)
		if err != nil {
			log.Fatal(err)
		}

		data, err := libmbd.Read(fileName)
		if err != nil {
			log.Fatal(err)
		}

		err = analyze(data, seed, numPulsesFlag, sytEnergyFlag, yEnergyFlag)
		if err != nil {
			log.Fatal(err)
		}
	}
}
