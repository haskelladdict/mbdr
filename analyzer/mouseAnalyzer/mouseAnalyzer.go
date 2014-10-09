// mouseAnalyzer determines vesicle release events and latencies for our
// mouse NMJ 6 AZ model with two synaptic vesicles each according to the
// second sensor faciliation model (see Ma et al., J. Neurophys, 2014)
package main

import (
	"flag"
	"fmt"
	"github.com/haskelladdict/mbdr/libmbd"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	numPulses      int     // number of AP pulses
	energyModel    bool    // use the energy model
	sytEnergy      int     // energy of activated synaptotagmin toward vesicle fusion
	yEnergy        int     // energy of activated Y sites toward vesicle fusion
	numActiveSites int     // number of simultaneously active sites required for release
	isiValue       float64 // interstimulus interval
)

func init() {
	flag.IntVar(&numPulses, "p", 1, "number of AP pulses in the model")
	flag.IntVar(&sytEnergy, "s", -1, "energy of active synaptotagmin sites "+
		"(required with -e flag)")
	flag.IntVar(&yEnergy, "y", -1, "energy of active y sites "+
		"(required with -e flag")
	flag.BoolVar(&energyModel, "e", false, "use the energy model instead of "+
		"deterministic model")
	flag.IntVar(&numActiveSites, "n", 0, "number of sites required for activation"+
		"for deterministic model")
	flag.Float64Var(&isiValue, "i", -1.0, "pulse duration in [s] for analysis multi "+
		"pulse data")
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

// printReleases prints a summary statistic for all released vesicle for a
// given seed
func printReleases(data *libmbd.MCellData, seed int, rel []*ReleaseEvent) {
	fmt.Println("mouseAnalyzer ran on", time.Now())
	if host, err := os.Hostname(); err == nil {
		fmt.Println("on ", host)
	}
	fmt.Println("\n-------------- parameters --------------")
	fmt.Println("number of pulses       : ", numPulses)
	fmt.Println("ISI                    : ", isiValue)
	if energyModel {
		fmt.Println("model                  : energy model")
		fmt.Println("syt energy             : ", sytEnergy)
		fmt.Println("y energy               : ", yEnergy)
	} else {
		fmt.Println("model                  : deterministic model")
		fmt.Println("number of active sites : ", numActiveSites)
	}
	fmt.Println("------------- data --------------------\n")
	timeStep := data.StepSize()
	for _, r := range rel {

		channels, err := determineCaChanContrib(data, r)
		if err != nil {
			log.Fatal(err)
		}

		eventTime := float64(r.eventIter) * timeStep
		pulseID := int(math.Floor(eventTime/isiValue)) + 1

		fmt.Printf("seed : %d   AZ : %d   ves : %d   time : %e   pulseID : %d", seed,
			r.azId+1, r.vesicleID+1, eventTime, pulseID)
		fmt.Printf("  sensors: [")
		for _, s := range r.sensors {
			fmt.Printf("%d ", s)
		}
		fmt.Printf("]")
		fmt.Printf("  channels: [")
		for n, c := range channels {
			fmt.Printf("%s : %d  ", n, int(c))
		}
		fmt.Printf("]")
		fmt.Printf("\n")
	}
}

// determineCaContrib determines which Ca channels contributed to the release
// of a particular vesicle.
// NOTE: We try to be as agnostic as we can in terms of the particular
// nomenclature used for naming the channels. However, the expectation is
// that data files tracking Ca binding to vesicles are named
// vesicle_<az>_<1|2>_ca_<ca naming>.<seed>.dat
func determineCaChanContrib(data *libmbd.MCellData, rel *ReleaseEvent) (map[string]float64, error) {
	channels := make(map[string]float64)
	// the az/channel counting is 1 based whereas our internal counting is 0 based
	regexString := fmt.Sprintf("vesicle_%d_%d_ca_.*", rel.azId+1, rel.vesicleID+1)
	counts, err := data.BlockDataByRegex(regexString)
	if err != nil {
		return nil, err
	}
	for k, c := range counts {
		if len(c.Col) != 1 {
			return nil, fmt.Errorf("data set %s has more than the expected 1 column",
				k)
		}
		if c.Col[0][rel.eventIter] > 0 {
			// need to subtract 2 from regexString due to the extra ".*"
			caString, err := extractCaChanName(k, len(regexString)-2)
			if err != nil {
				return nil, err
			}
			channels[caString] = c.Col[0][rel.eventIter]
		}
	}

	return channels, nil
}

// extractCaChanName attempts to extract the name of the calcium channel based
// on the expected data name pattern vesicle_<az>_<1|2>_ca_<ca naming>.<seed>.dat
func extractCaChanName(name string, prefixLength int) (string, error) {
	caName := name[prefixLength:]
	items := strings.Split(caName, ".")
	if len(items) == 0 {
		return "", fmt.Errorf("Could not determine Ca channel name from data set %s", name)
	}
	return items[0], nil
}

// main entry point
func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		usage()
		return
	}

	// some sanity checks
	if energyModel && (sytEnergy < 0 || yEnergy < 0) {
		log.Fatal("Please provide a non-negative synaptotagmin and y site energy")
	}

	if !energyModel && numActiveSites == 0 {
		log.Fatal("Please provide a positive count for the number of required active sites")
	}

	if numPulses > 1 && isiValue <= 0 {
		log.Fatal("Analysis multi-pulse data requires a non-zero ISI value.")
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

		err = analyze(data, energyModel, seed, numPulses, numActiveSites, sytEnergy,
			yEnergy)
		if err != nil {
			log.Fatal(err)
		}
	}
}
