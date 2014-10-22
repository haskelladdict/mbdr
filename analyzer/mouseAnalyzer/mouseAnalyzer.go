// mouseAnalyzer determines vesicle release events and latencies for our
// mouse NMJ 6 AZ model with two synaptic vesicles each according to the
// the original excess-calcium-binding-site model
// (Dittrich et al., Biophys. J, 2013, 104:2751-2763)
package main

import (
	"flag"
	"fmt"
	rel "github.com/haskelladdict/mbdr/analyzer/releaser"
)

// simulation model
var model = rel.SimModel{
	VesicleIDs: []string{"1_1", "1_2", "2_1", "2_2", "3_1", "3_2", "4_1", "4_2",
		"5_1", "5_2", "6_1", "6_2"},
	SensorTemplate: "bound_vesicle_%s_%s_%d.%04d.dat",
	PulseDuration:  3e-3,
	NumPulses:      1,
}

// fusion model
var fusionModel = rel.FusionModel{
	NumSyt:       8,
	NumActiveSyt: 2,
	EnergyModel:  false,
}

// number of threads
var numThreads int

// initialize simulation and fusion model parameters coming from commandline
func init() {

	flag.IntVar(&fusionModel.NumActiveSites, "n", 0, "number of sites required for activation "+
		"of deterministic model")
	flag.IntVar(&numThreads, "T", 1, "number of threads. Each thread works on a "+
		"single binary output file\n\tso memory requirements multiply")

	// define synaptogamin and Y sites
	model.CaSensors = make([]rel.CaSensor, fusionModel.NumSyt+fusionModel.NumY)
	model.CaSensors[0] = rel.CaSensor{[]int{8, 9, 29, 30, 31}, rel.SytSite}
	model.CaSensors[1] = rel.CaSensor{[]int{7, 32, 33, 34, 35}, rel.SytSite}
	model.CaSensors[2] = rel.CaSensor{[]int{3, 6, 36, 37, 38}, rel.SytSite}
	model.CaSensors[3] = rel.CaSensor{[]int{17, 39, 40, 41, 42}, rel.SytSite}
	model.CaSensors[4] = rel.CaSensor{[]int{15, 16, 43, 44, 45}, rel.SytSite}
	model.CaSensors[5] = rel.CaSensor{[]int{14, 46, 47, 48, 49}, rel.SytSite}
	model.CaSensors[6] = rel.CaSensor{[]int{4, 12, 24, 50, 51}, rel.SytSite}
	model.CaSensors[7] = rel.CaSensor{[]int{10, 25, 26, 27, 28}, rel.SytSite}
}

// usage prints a brief usage information to stdout
func usage() {
	fmt.Println("usage: mouseAnalyzer [options] <binary mcell files>")
	fmt.Println("\noptions:")
	flag.PrintDefaults()
}

// main entry point
func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		usage()
		return
	}

	rel.Run(&model, &fusionModel, flag.Args(), numThreads)
}
