// Package releaser determines vesicle release events and latencies for our
// from and mouse NMJ AZ models.
package releaser

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/haskelladdict/mbdr/libmbd"
	"github.com/haskelladdict/mbdr/parser"
	"github.com/haskelladdict/mbdr/version"
)

// extractSeed attempts to extract the seed from the filename of the provided
// binary mcell data file.
// NOTE: the following filenaming convention is assumed *.<seedIDString>.bin.(gz|bz2)
func extractSeed(fileName string) (int, error) {
	items := strings.Split(fileName, ".")
	if len(items) <= 3 {
		return -1, fmt.Errorf("incorrectly formatted fileName %s. "+
			"Expected *.<seedIDString>.bin.(gz|bz2)", fileName)
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
	return -1, fmt.Errorf("Unable to extract seed id from filename %s", fileName)
}

// printHeader prints and informative header file with date and commandline
// options requested for analysis
func printHeader(model *SimModel, fusion *FusionModel, info *AnalyzerInfo) {
	fmt.Printf("%s v%s ran on %s\n", info.Name, version.Tag, time.Now())
	if host, err := os.Hostname(); err == nil {
		fmt.Println("on ", host)
	}
	fmt.Println("\n-------------- parameters --------------")
	fmt.Println("number of pulses       :", model.NumPulses)
	if model.NumPulses > 1 {
		fmt.Println("ISI                    :", model.IsiValue, "s")
	}
	if fusion.EnergyModel {
		fmt.Println("model                  : energy model")
		fmt.Println("syt energy             :", fusion.SytEnergy)
		fmt.Println("y energy               :", fusion.YEnergy)
	} else {
		fmt.Println("model                  : deterministic model")
		fmt.Println("number of active sites :", fusion.NumActiveSites)
	}
	fmt.Println("-------------- data --------------------")
	fmt.Println("")
}

// determineCaContrib determines which Ca channels contributed to the release
// of a particular vesicle.
// NOTE: We try to be as agnostic as we can in terms of the particular
// nomenclature used for naming the channels. However, the expectation is
// that data files tracking Ca binding to vesicles are named
// vesicle_<az>_<1|2>_ca_<ca naming>.<seed>.dat for syt, and
// vesicle_Y_<az>_<1|2>_ca_<ca naming>.<seed>.dat for Y.
func determineCaChanContrib(data *libmbd.MCellData, rel *ReleaseEvent) (map[string]float64, error) {
	channels := make(map[string]float64)
	regexString := fmt.Sprintf("vesicle(_Y)?_%s_ca_.*", rel.vesicleID)
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
			subs := strings.SplitAfter(k, "ca_")
			if len(subs) < 2 {
				return nil, fmt.Errorf("could not determined Ca channel name")
			}
			caString, err := extractCaChanName(subs[1])
			if err != nil {
				return nil, err
			}
			channels[caString] += c.Col[0][rel.eventIter]
		}
	}

	return channels, nil
}

// extractCaChanName attempts to extract the name of the calcium channel based
// on the expected data name pattern <ca naming>.<seed>.dat
func extractCaChanName(name string) (string, error) {
	items := strings.Split(name, ".")
	if len(items) == 0 {
		return "", fmt.Errorf("Could not determine Ca channel name from data set %s", name)
	}
	return items[0], nil
}

// createAnalysisJobs fills a channel with binary data filenames to be analyzed
func createAnalysisJobs(fileNames []string, analysisJobs chan<- string) {
	for _, n := range fileNames {
		analysisJobs <- n
	}
	close(analysisJobs)
}

// runJob is responsible for analyzing the data files provided in the
// analysisJob channel
func runJob(analysisJobs <-chan string, done chan<- []string, m *SimModel,
	f *FusionModel, errMsgs chan<- string) {

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for fileName := range analysisJobs {
		seed, err := extractSeed(fileName)
		if err != nil {
			errMsgs <- fmt.Sprintln(err)
			done <- nil
			continue
		}

		data, err := parser.Read(fileName)
		if err != nil {
			errMsgs <- fmt.Sprintln(err)
			done <- nil
			continue
		}

		releaseMsgs, err := analyze(data, m, f, rng, seed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to analyze output file %s: %s\n", fileName, err)
			done <- nil
			continue
		}
		// NOTE: This is a bit of a hack but since we're dealing with potentially
		// large data sets we need to make sure to free memory before we start
		// working on the next one
		debug.FreeOSMemory()

		done <- releaseMsgs
	}
}

// Run is the main entry point for the release analysis and spawns the
// requested number of analysis goroutines
func Run(model *SimModel, fusion *FusionModel, info *AnalyzerInfo, args []string) {
	// some sanity checks
	if fusion.EnergyModel && (fusion.SytEnergy < 0 || fusion.YEnergy < 0) {
		log.Fatal("Please provide a non-negative synaptotagmin and y site energy")
	}

	if !fusion.EnergyModel && fusion.NumActiveSites == 0 {
		log.Fatal("Please provide a positive count for the number of required active sites")
	}

	if model.NumPulses > 1 && model.IsiValue <= 0 {
		log.Fatal("Analysis multi-pulse data requires a non-zero ISI value.")
	}

	// request proper number of go routines.
	runtime.GOMAXPROCS(info.NumThreads)

	printHeader(model, fusion, info)
	analysisJobs := make(chan string)
	go createAnalysisJobs(args, analysisJobs)

	done := make(chan []string)
	errMsgs := make(chan string)
	for i := 0; i < info.NumThreads; i++ {
		go runJob(analysisJobs, done, model, fusion, errMsgs)
	}

	// collect all errors
	var errors []string
	go func() {
		for m := range errMsgs {
			errors = append(errors, m)
		}
	}()

	for i := 0; i < len(args); i++ {
		msgs := <-done
		for _, m := range msgs {
			fmt.Println(m)
		}
	}
	close(errMsgs)

	// print errors
	if len(errors) != 0 {
		fmt.Println("\n\n------------------------------------------")
		fmt.Printf("ERROR: %d output files could not be processed!\n", len(errors))
		fmt.Println("\nReason:")
		for _, e := range errors {
			fmt.Print(e)
		}
	}
}
