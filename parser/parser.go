// parser is a wrapper around the main parsing routines. It figures out the
// API version of the underlying data and then dispatches the proper parser.
package parser

import (
	"compress/bzip2"
	"github.com/haskelladdict/mbdr/libmbd"
	"github.com/haskelladdict/mbdr/parser/parseAPI2"
	"io"
	"os"
)

const apiTagLength = len("MCELL_BINARY_API_2")

// ReadHeader opens the binary mcell data file and parses the header without
// reading the actual data. This provides efficient access to metadata and
// the names of stored data blocks. After calling this function the buffer
// field of MCellData is set to nil since no data is parsed.
func ReadHeader(filename string) (*libmbd.MCellData, error) {
	fileRaw, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fileRaw.Close()
	file := bzip2.NewReader(fileRaw)

	// check API version and pick proper reader
	apiTag, err := parseAPITag(file)
	if err != nil {
		return nil, err
	}

	data := new(libmbd.MCellData)
	switch apiTag {
	case "MCELL_BINARY_API_2":
		if data, err = parseAPI2.Header(file, data); err != nil {
			return nil, err
		}
	}

	return data, nil

}

// Read header opens the binary mcell data file and parses the header and the
// actual data stored. If only access to the metadata is required, it is much
// more efficient to only call ReadHeader directly.
func Read(filename string) (*libmbd.MCellData, error) {
	fileRaw, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fileRaw.Close()
	file := bzip2.NewReader(fileRaw)

	// check API version and pick proper reader
	apiTag, err := parseAPITag(file)
	if err != nil {
		return nil, err
	}

	data := new(libmbd.MCellData)
	switch apiTag {
	case "MCELL_BINARY_API_2":
		if data, err = parseAPI2.Header(file, data); err != nil {
			return nil, err
		}
		if data, err = parseAPI2.Data(file, data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

// parseAPITag reads the API tag inside the data set
func parseAPITag(r io.Reader) (string, error) {
	receivedAPITag := make([]byte, apiTagLength)
	if _, err := io.ReadFull(r, receivedAPITag); err != nil {
		return "", err
	}
	return string(receivedAPITag), nil
}
