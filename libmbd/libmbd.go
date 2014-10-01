package libmbd

import (
	"compress/bzip2"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math"
	"os"
)

const requiredAPITag = "MCELL_BINARY_API_2"

// convenience consts describing the length of certain C types in bytes
const (
	len_uint16_t = 2
	len_uint64_t = 8
	len_double   = 8
)

// enumerations defining the output type of the included data
const (
	Step uint16 = 1 << iota
	TimeList
	IterationList
)

// blockData stores metadata related to a given data block such as the
// data name, number of data columns, the type of data stored (int/double),
// and the internal offset into the buffer at which the data can be found.
type blockData struct {
	name      string
	numCols   uint64
	dataTypes []uint16
	offset    uint64
}

// MCellData tracks the data contained in the binary mcell file as well as
// relevant metadate to retrieve specific data items.
type MCellData struct {
	buffer        readBuf
	outputType    uint16
	blockSize     uint64
	stepSize      float64
	timeList      []float64
	outputBufSize uint64
	numBlocks     uint64
	blockNames    []string
	blockNameMap  map[string]uint64
	blockInfo     []blockData
}

// BlockNames returns the list of available blocknames
func (d *MCellData) BlockNames() []string {
	return d.blockNames
}

// NumDataBlocks returns the number of available datablocks
func (d *MCellData) NumDataBlocks() uint64 {
	return d.numBlocks
}

// BlockSize returns the number of items per datablock
func (d *MCellData) BlockSize() uint64 {
	return d.blockSize
}

// OutputType returns the output type (STEP, ITERATION_LIST/TIME_LIST)
func (d *MCellData) OutputType() uint16 {
	return d.outputType
}

// StepSize returns the output step size. NOTE: The returns value is only
// meaningful is outputType == Step, otherwise this function returns 0
func (d *MCellData) StepSize() float64 {
	return d.stepSize
}

// Read opens the binary mcell data file and attempts to parse its content
// into a Data buffer struct.
func Read(filename string) (*MCellData, error) {
	fileRaw, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fileRaw.Close()
	file := bzip2.NewReader(fileRaw)

	var data MCellData
	data.buffer, err = ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	err = parseHeader(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

// checkAPITag reads the API tag inside the data set and makes sure it
// matches the required version
func checkAPITag(data *MCellData) error {
	receivedAPITag := string(data.buffer[:len(requiredAPITag)])
	if receivedAPITag != requiredAPITag {
		return fmt.Errorf("incorrect API Tag %s", receivedAPITag)
	}

	data.buffer = data.buffer[len(requiredAPITag)+1:]
	return nil
}

// parseBlockInfo reads the pertinent data block information such as the
// time step, time list, number of data blocks etc.
func parseBlockInfo(data *MCellData) error {

	data.outputType = data.buffer.uint16()
	data.blockSize = data.buffer.uint64()

	length := data.buffer.uint64()
	switch data.outputType {
	case Step:
		data.stepSize = data.buffer.float64()

	case TimeList:
		for i := uint64(0); i < length; i++ {
			data.timeList = append(data.timeList, data.buffer.float64())
		}

	case IterationList:
		for i := uint64(0); i < length; i++ {
			data.timeList = append(data.timeList, data.buffer.float64())
		}

	default:
		return fmt.Errorf("encountered unknown data output type")
	}

	data.outputBufSize = data.buffer.uint64()
	data.numBlocks = data.buffer.uint64()

	return nil
}

// parseBlockNames extract the names of data blocks contained with the data file
func parseBlockNames(data *MCellData) error {

	// initialize blockname map
	data.blockNameMap = make(map[string]uint64)

	var offset uint64
	for count := uint64(0); count < data.numBlocks; count++ {

		e := blockData{}
		e.offset = offset
		var i int

		// find end of next block name
		for string(data.buffer[i]) != "\x00" {
			i++
		}
		e.name = string(data.buffer[:i])
		data.blockNames = append(data.blockNames, e.name)
		data.blockNameMap[e.name] = count
		data.buffer = data.buffer[i+1:]

		e.numCols = data.buffer.uint64()
		offset += e.numCols
		for c := uint64(0); c < e.numCols; c++ {
			e.dataTypes = append(e.dataTypes, data.buffer.uint16())
		}
		data.blockInfo = append(data.blockInfo, e)
	}

	return nil
}

// parseHeader reads the header of the binary mcell data file to check the API
// version and retrieve general information regarding the data contained
// within (number of datablocks, block names, ...).
func parseHeader(data *MCellData) error {

	if err := checkAPITag(data); err != nil {
		return err
	}

	if err := parseBlockInfo(data); err != nil {
		return err
	}

	if err := parseBlockNames(data); err != nil {
		return err
	}

	return nil
}

// readBuf and helper function convert between a byte slice and an underlying
// integer type
// NOTE: This code was take almost verbatim from archive/zip/reader from the
// standard library
type readBuf []byte

func (b *readBuf) uint16() uint16 {
	v := binary.LittleEndian.Uint16(*b)
	*b = (*b)[2:]
	return v
}

func (b *readBuf) uint32() uint32 {
	v := binary.LittleEndian.Uint32(*b)
	*b = (*b)[4:]
	return v
}

func (b *readBuf) uint64() uint64 {
	v := binary.LittleEndian.Uint64(*b)
	*b = (*b)[8:]
	return v
}

func (b *readBuf) float64() float64 {
	v := math.Float64frombits(binary.LittleEndian.Uint64(*b))
	*b = (*b)[8:]
	return v
}
