// frogAnalyzer determines vesicle release events and latencies for our
// frog NMJ model according to the second sensor facilitation model
// (see Ma et al., J. Neurophys, 2014)
package main

import (
	"flag"
	"fmt"
	"os"

	rel "github.com/haskelladdict/mbdr/releaser"
	"github.com/haskelladdict/mbdr/version"
)

// analyser info
var info = rel.AnalyzerInfo{
	Name: "frogAnalyzerY",
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
	SensorTemplate: "bound_vesicle_%s_%s_%02d_%d.%04d.dat",
	PulseDuration:  3e-3,
}

// fusion model
var fusionModel = rel.FusionModel{
	NumSyt:              8,
	NumY:                16,
	NumActiveSyt:        2,
	NumActiveY:          1,
	VesicleFusionEnergy: 40,
}

// initialize simulation and fusion model parameters coming from commandline
func init() {

	flag.IntVar(&model.NumPulses, "p", 1, "number of AP pulses in the model")
	flag.IntVar(&fusionModel.SytEnergy, "s", -1, "energy of active synaptotagmin sites "+
		"(required with -e flag)")
	flag.IntVar(&fusionModel.YEnergy, "y", -1, "energy of active y sites "+
		"(required with -e flag")
	flag.BoolVar(&fusionModel.EnergyModel, "e", false, "use the energy model instead of "+
		"deterministic model")
	flag.IntVar(&fusionModel.NumActiveSites, "n", 0, "number of sites required for activation "+
		"of deterministic model")
	flag.Float64Var(&model.IsiValue, "i", -1.0, "pulse interval in [s] for analysis multi "+
		"pulse data (requires p > 1)")
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
	model.CaSensors[8] = rel.CaSensor{[]int{122}, rel.YSite}
	model.CaSensors[9] = rel.CaSensor{[]int{70}, rel.YSite}
	model.CaSensors[10] = rel.CaSensor{[]int{126}, rel.YSite}
	model.CaSensors[11] = rel.CaSensor{[]int{142}, rel.YSite}
	model.CaSensors[12] = rel.CaSensor{[]int{62}, rel.YSite}
	model.CaSensors[13] = rel.CaSensor{[]int{118}, rel.YSite}
	model.CaSensors[14] = rel.CaSensor{[]int{22}, rel.YSite}
	model.CaSensors[15] = rel.CaSensor{[]int{134}, rel.YSite}
	model.CaSensors[16] = rel.CaSensor{[]int{110}, rel.YSite}
	model.CaSensors[17] = rel.CaSensor{[]int{66}, rel.YSite}
	model.CaSensors[18] = rel.CaSensor{[]int{106}, rel.YSite}
	model.CaSensors[19] = rel.CaSensor{[]int{130}, rel.YSite}
	model.CaSensors[20] = rel.CaSensor{[]int{2}, rel.YSite}
	model.CaSensors[21] = rel.CaSensor{[]int{114}, rel.YSite}
	model.CaSensors[22] = rel.CaSensor{[]int{42}, rel.YSite}
	model.CaSensors[23] = rel.CaSensor{[]int{138}, rel.YSite}
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

	if model.IsiValue != 0.0 && model.NumPulses <= 1 {
		fmt.Fprintf(os.Stderr, "ERROR: Specification of an pulse interval requires p > 1\n\n")
		usage()
		return
	}

	rel.Run(&model, &fusionModel, &info, flag.Args())
}
