package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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
)

func init() {
	flag.BoolVar(&infoFlag, "i", false, "show general info")
	flag.BoolVar(&listFlag, "l", false, "list available data blocks")
	flag.BoolVar(&extractFlag, "e", false, "extract dataset")
	flag.BoolVar(&addTimesFlag, "t", false, "add output times column")
	flag.BoolVar(&writeFileFlag, "w", false, "write output to file")
	flag.Uint64Var(&extractID, "I", 0, "id of dataset to extract")
	flag.StringVar(&extractString, "N", "", "name of dataset to extract")
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
		writeData(data)
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

// writeData extracts the content of a data set either at the requested ID
// or the provided name and writes it to stdout or a file if requested
// NOTE: This routine doesn't bother with converting to integer column data
// (as determined by DataTypes) and simply prints everything as double
func writeData(d *libmbd.MCellData) {
	var data *libmbd.CountData
	name := extractString
	var err error
	if extractString != "" {
		if data, err = d.BlockDataByName(extractString); err != nil {
			log.Fatal(err)
		}
	} else {
		if data, err = d.BlockDataByID(extractID); err != nil {
			log.Fatal(err)
		}
		if name, err = d.IDtoBlockName(extractID); err != nil {
			log.Fatal(err)
		}
	}

	var outputTimes []float64
	if addTimesFlag {
		outputTimes = d.OutputTimes()
	}

	output := os.Stdout
	if writeFileFlag {
		if output, err = os.Create(name); err != nil {
			log.Fatal(err)
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
}
