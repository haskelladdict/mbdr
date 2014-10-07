package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/haskelladdict/mbdr/libmbd"
)

const mbdrVersion = 2

// command line flags
var (
	infoFlag      bool
	listFlag      bool
	addTimesFlag  bool
	writeFileFlag bool
	extractFlag   bool
	extractID     uint64
	extractString string
	extractRegex  string
)

func init() {
	flag.BoolVar(&infoFlag, "i", false, "show general info")
	flag.BoolVar(&listFlag, "l", false, "list available data blocks")
	flag.BoolVar(&extractFlag, "e", false, "extract dataset")
	flag.BoolVar(&addTimesFlag, "t", false, "add output times column")
	flag.BoolVar(&writeFileFlag, "w", false, "write output to file")
	flag.Uint64Var(&extractID, "I", 0, "id of dataset to extract")
	flag.StringVar(&extractString, "N", "", "name of dataset to extract")
	flag.StringVar(&extractRegex, "R", "", "regular expression of dataset(s) to extract")
}

// main function entry point
func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		usage()
		return
	}

	filename := flag.Args()[0]
	var data *libmbd.MCellData
	var err error
	if infoFlag || listFlag {
		if data, err = libmbd.ReadHeader(filename); err != nil {
			log.Fatal(err)
		}
	} else if extractFlag {
		if data, err = libmbd.Read(filename); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("\nError: Please specify at least one of -i, -l, or -e!\n")
		usage()
		return
	}

	switch {
	case infoFlag:
		showInfo(data)

	case listFlag:
		showAvailableData(data)

	case extractFlag:
		if err := extractData(data); err != nil {
			log.Fatal(err)
		}
	}
}

// usage prints a brief usage information to stdout
func usage() {
	fmt.Println("usage: mbdr [options] <binary mcell filename>")
	fmt.Println("\noptions:")
	flag.PrintDefaults()
}

// showInfo provides general info regarding the nature and amount of data
// contained in the binary mcell file
func showInfo(d *libmbd.MCellData) {
	fmt.Printf("This is mbdr version %d        (C) 2014 M. Dittrich\n", mbdrVersion)
	fmt.Println("--------------------------------------------------")
	fmt.Printf("mbdr> found %d datablocks with %d items each\n", d.NumDataBlocks(),
		d.BlockSize())
	switch d.OutputType() {
	case libmbd.Step:
		fmt.Printf("mbdr> output generated via STEP size of %g s\n", d.StepSize())

	case libmbd.TimeList:
		fmt.Printf("mbdr> output generated via TIME_LIST\n")

	case libmbd.IterationList:
		fmt.Printf("mbdr> output generated via ITERATION_LIST\n")

	default:
		fmt.Printf("mbdr> encountered UNKNOWN output type")
	}
}

// showAvailableData shows the available data sets contained in the
// binary output file
func showAvailableData(d *libmbd.MCellData) {
	for i, n := range d.BlockNames() {
		fmt.Printf("[%d] %s\n", i, n)
	}
}

// extractData extracts the content of a data set or data sets either at the
// requested ID, the provided name, or the regular expression and writes it to
// stdout or files if requested.
// NOTE: This routine doesn't bother with converting to integer column data
// (as determined by DataTypes) and simply prints everything as double
func extractData(data *libmbd.MCellData) error {

	outputData := make(map[string]*libmbd.CountData)
	var countData *libmbd.CountData
	var err error

	if extractString != "" {
		// if match string was supplied we'll use it
		if countData, err = data.BlockDataByName(extractString); err != nil {
			return err
		}
		outputData[extractString] = countData
	} else if extractRegex != "" {
		// if a regexp string was supplied we try to compile it and then determine
		// all matches against available data set names
		regex, err := regexp.Compile(extractRegex)
		if err != nil {
			return err
		}
		names := data.BlockNames()
		for _, n := range names {
			if regex.MatchString(n) {
				if countData, err = data.BlockDataByName(n); err != nil {
					return err
				}
				outputData[n] = countData
			}
		}
	} else {
		// otherwise we pick the supplied data set ID to extract (0 by default)
		if countData, err = data.BlockDataByID(extractID); err != nil {
			return err
		}
		var name string
		if name, err = data.IDtoBlockName(extractID); err != nil {
			return err
		}
		outputData[name] = countData
	}

	for name, col := range outputData {
		if err = writeData(data, name, col); err != nil {
			return err
		}
	}

	return nil
}

// writeData writes the supplied count data corresponding to the named data set
// to stdout or a file
func writeData(d *libmbd.MCellData, name string, data *libmbd.CountData) error {

	var outputTimes []float64
	if addTimesFlag {
		outputTimes = d.OutputTimes()
	}

	output := os.Stdout
	var err error
	if writeFileFlag {
		if output, err = os.Create(name); err != nil {
			return err
		}
	}

	numCols := len(data.Col)
	numRows := len(data.Col[0])
	for r := 0; r < numRows; r++ {
		for c := 0; c < numCols; c++ {
			if addTimesFlag {
				fmt.Fprintf(output, "%8.5e %g", outputTimes[r], data.Col[c][r])
			} else {
				fmt.Fprintf(output, "%g", data.Col[c][r])
			}
		}
		fmt.Fprintf(output, "\n")
	}
	return nil
}
