// mouseAnalyzer determines vesicle release events and latencies for our
// mouse NMJ 6 AZ model with two synaptic vesicles each according to the
// second sensor faciliation model (see Ma et al., J. Neurophys, 2014)
package main

import (
	"github.com/haskelladdict/mbdr/analyzer/releaseAnalyzer"
)

// list of vesicle IDs for mouse model
var vesicleIDs = []string{"1_1", "1_2", "2_1", "2_2", "3_1", "3_2", "4_1", "5_1",
	"5_2", "6_1", "6_2"}

// main entry point
func main() {
	releaseAnalyzer.Run(vesicleIDs)
}
