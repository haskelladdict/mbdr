package main

import (
	"fmt"
	"github.com/haskelladdict/mbdr/libmbd"
)

const (
	numAZ       = 6  // number of active zones (AZ) in the model
	numVesicles = 2  // number of vesicles per AZ
	numSyt      = 8  // number of synaptotagmin molecules (with 5 Ca2+ sites each)
	numY        = 16 // number of second sensor (Y) sites
)

const vesicleFusionEnergy = 40

// CaSite defines synaptotagmin and Y sites
type caSite struct {
	sites []int
}

var caSites []caSite

func init() {
	caSites = make([]caSite, numSyt+numY)
}

func analyze(data *libmbd.MCellData, numPulsesFlag, sytEnergyFlag,
	yEnergyFlag int) error {
	fmt.Println("foo", len(caSites))
	return nil
}
