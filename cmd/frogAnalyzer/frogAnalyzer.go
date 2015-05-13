// frogAnalyzer determines vesicle release events and latencies for our
// frog NMJ model according to the original excess-calcium-binding-site
// model (Dittrich et al., Biophys. J, 2013, 104:2751-2763)
package main

import (
	"flag"
	"fmt"

	rel "github.com/haskelladdict/mbdr/releaser"
	"github.com/haskelladdict/mbdr/version"
)

// analyser info
var info = rel.AnalyzerInfo{
	Name: "frogAnalyzer",
}

// simulation model
var model = rel.SimModel{
	VesicleIDs: []string{"01", "02", "03", "04", "05", "06", "07", "08", "09",
		"10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22",
		"23", "24", "25", "26"},
	VGCCVesicleMap: map[string]string{"01": "A01", "02": "A04", "03": "A09",
		"04": "A13", "05": "A17", "06": "A21", "07": "A25", "08": "A29",
		"09": "A33", "10": "A38", "11": "A42", "12": "A46", "13": "A49",
		"14": "D01", "15": "D04", "16": "D09", "17": "D13", "18": "D17",
		"19": "D21", "20": "D25", "21": "D29", "22": "D33", "23": "D37",
		"24": "D42", "25": "D46", "26": "D49",
	},
	SensorTemplate: "bound_vesicle_%s_%s_%02d.%04d.dat",
	PulseDuration:  3e-3,
	NumPulses:      1,
}

// fusion model
var fusionModel = rel.FusionModel{
	NumSyt:       8,
	NumActiveSyt: 2,
	EnergyModel:  false,
}

// initialize simulation and fusion model parameters coming from commandline
func init() {

	flag.IntVar(&fusionModel.NumActiveSites, "n", 0, "number of sites required for activation "+
		"of deterministic model")
	flag.IntVar(&info.NumThreads, "T", 1, "number of threads. Each thread works on a "+
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
	fmt.Printf("%s v%s  (C) %s Markus Dittrich\n\n", info.Name, version.Tag, version.Year)
	fmt.Printf("usage: %s [options] <binary mcell files>\n", info.Name)
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

	rel.Run(&model, &fusionModel, &info, flag.Args())
}
