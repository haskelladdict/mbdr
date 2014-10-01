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

func init() {
	flag.BoolVar(&infoFlag, "i", false, "show general info")
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
	}
	/*
		for i, s := range data.GetBlockNames() {
			fmt.Println(i, s)
		}
	*/
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
