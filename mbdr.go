package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/haskelladdict/mbdr/libmbd"
)

const mbdrVersion = 2

// command line flags
var infoFlag bool
var listFlag bool
var extractIDFlag int

func init() {
	flag.BoolVar(&infoFlag, "i", false, "show general info")
	flag.BoolVar(&listFlag, "l", false, "list available data blocks")
	flag.IntVar(&extractIDFlag, "I", -1, "extract dataset at given index")
}

// main function entry point
func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		usage()
		return
	}

	filename := flag.Args()[0]
	data, err := libmbd.Read(filename)
	if err != nil {
		log.Fatal(err)
	}

	switch {
	case infoFlag:
		showInfo(data)

	case listFlag:
		showAvailableData(data)

	case extractIDFlag >= 0:
		showDataByID(data, extractIDFlag)
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

// showDataByID prints the content of the data set at the requested ID
// NOTE: This routine doesn't bother with converting integer column data and
// simply prints it as double
func showDataByID(d *libmbd.MCellData, id int) {
	data, err := d.BlockDataByID(uint64(id))
	if err != nil {
		log.Fatal(err)
	}

	numCols := len(data.Col)
	numRows := len(data.Col[0])
	for r := 0; r < numRows; r++ {
		for c := 0; c < numCols; c++ {
			fmt.Printf("%g", data.Col[c][r])
		}
		fmt.Printf("\n")
	}
}
