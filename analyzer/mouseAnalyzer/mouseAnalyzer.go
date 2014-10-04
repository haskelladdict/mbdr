// mouseAnalyzer determines vesicle release events and latencies for our
// mouse NMJ 6 AZ model with two synaptic vesicles each according to the
// second sensor faciliation model (see Ma et al., J. Neurophys, 2014)
package main

import (
	"flag"
	"fmt"
	"github.com/haskelladdict/mbdr/libmbd"
	"log"
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

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		usage()
		return
	}
	if sytEnergyFlag < 0 || yEnergyFlag < 0 {
		log.Fatal("Please provide a non-negative synaptotagmin and y site energy")
	}

	filename := flag.Args()[0]
	data, err := libmbd.Read(filename)
	if err != nil {
		log.Fatal(err)
	}

	if err := analyze(data, numPulsesFlag, sytEnergyFlag, yEnergyFlag); err != nil {
		log.Fatal(err)
	}
}
