// mouseAnalyzer determines vesicle release events and latencies for our
// mouse NMJ 6 AZ model with two synaptic vesicles each according to the
// second sensor faciliation model (see Ma et al., J. Neurophys, 2014)
package main

import (
	"flag"
	"fmt"
	"os"

	rel "github.com/haskelladdict/mbdr/releaser"
	"github.com/haskelladdict/mbdr/version"
)

// simulation model
var model = rel.SimModel{
	VesicleIDs: []string{"1_1", "1_2", "2_1", "2_2", "3_1", "3_2", "4_1", "4_2",
		"5_1", "5_2", "6_1", "6_2"},
	SensorTemplate: "bound_vesicle_%s_%s_%d_%d.%04d.dat",
	PulseDuration:  3e-3,
}

// analyser info
var info = rel.AnalyzerInfo{
	Name: "mouseAnalyzerY",
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

	flag.IntVar(&model.NumPulses, "p", 2, "number of AP pulses in the model")
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
	model.CaSensors[0] = rel.CaSensor{Sites: []int{8, 9, 29, 30, 31}, SiteType: rel.SytSite}
	model.CaSensors[1] = rel.CaSensor{Sites: []int{7, 32, 33, 34, 35}, SiteType: rel.SytSite}
	model.CaSensors[2] = rel.CaSensor{Sites: []int{3, 6, 36, 37, 38}, SiteType: rel.SytSite}
	model.CaSensors[3] = rel.CaSensor{Sites: []int{17, 39, 40, 41, 42}, SiteType: rel.SytSite}
	model.CaSensors[4] = rel.CaSensor{Sites: []int{15, 16, 43, 44, 45}, SiteType: rel.SytSite}
	model.CaSensors[5] = rel.CaSensor{Sites: []int{14, 46, 47, 48, 49}, SiteType: rel.SytSite}
	model.CaSensors[6] = rel.CaSensor{Sites: []int{4, 12, 24, 50, 51}, SiteType: rel.SytSite}
	model.CaSensors[7] = rel.CaSensor{Sites: []int{10, 25, 26, 27, 28}, SiteType: rel.SytSite}
	model.CaSensors[8] = rel.CaSensor{Sites: []int{122}, SiteType: rel.YSite}
	model.CaSensors[9] = rel.CaSensor{Sites: []int{70}, SiteType: rel.YSite}
	model.CaSensors[10] = rel.CaSensor{Sites: []int{126}, SiteType: rel.YSite}
	model.CaSensors[11] = rel.CaSensor{Sites: []int{142}, SiteType: rel.YSite}
	model.CaSensors[12] = rel.CaSensor{Sites: []int{62}, SiteType: rel.YSite}
	model.CaSensors[13] = rel.CaSensor{Sites: []int{118}, SiteType: rel.YSite}
	model.CaSensors[14] = rel.CaSensor{Sites: []int{22}, SiteType: rel.YSite}
	model.CaSensors[15] = rel.CaSensor{Sites: []int{134}, SiteType: rel.YSite}
	model.CaSensors[16] = rel.CaSensor{Sites: []int{110}, SiteType: rel.YSite}
	model.CaSensors[17] = rel.CaSensor{Sites: []int{66}, SiteType: rel.YSite}
	model.CaSensors[18] = rel.CaSensor{Sites: []int{106}, SiteType: rel.YSite}
	model.CaSensors[19] = rel.CaSensor{Sites: []int{130}, SiteType: rel.YSite}
	model.CaSensors[20] = rel.CaSensor{Sites: []int{2}, SiteType: rel.YSite}
	model.CaSensors[21] = rel.CaSensor{Sites: []int{114}, SiteType: rel.YSite}
	model.CaSensors[22] = rel.CaSensor{Sites: []int{42}, SiteType: rel.YSite}
	model.CaSensors[23] = rel.CaSensor{Sites: []int{138}, SiteType: rel.YSite}
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

	if model.NumPulses <= 1 {
		fmt.Fprintf(os.Stderr, "ERROR: This analyzer requires p > 1\n\n")
		usage()
		return
	}

	rel.Run(&model, &fusionModel, &info, flag.Args())
}
