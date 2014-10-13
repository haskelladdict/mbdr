// frogAnalyzer determines vesicle release events and latencies for our
// frog NMJ model with two synaptic vesicles each according to the
// second sensor faciliation model (see Ma et al., J. Neurophys, 2014)
package main

import (
	"github.com/haskelladdict/mbdr/analyzer/releaseAnalyzer"
)

// list of vesicle IDs for mouse model
var vesicleIDs = []string{"01", "02", "03", "04", "05", "06", "07", "08", "09",
	"10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22",
	"23", "24", "25", "26"}

// template string for vesicle binding
const template = "bound_vesicle_%s_%s_%02d_%d.%04d.dat"

// main entry point
func main() {
	releaseAnalyzer.Run(&releaseAnalyzer.ReleaseModel{vesicleIDs, template})
}
