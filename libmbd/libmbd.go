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

// enumeration describing the output type of the included data
const (
	Step uint16 = 1 << iota
	TimeList
	IterationList
)

// enumeration describing the data type of a data column
const (
	IntData = iota
	DoubleData
)

// MCellData tracks the data contained in the binary mcell file as well as
// relevant metadate to retrieve specific data items.
type MCellData struct {
	buffer        readBuf
	outputType    uint16
	blockSize     uint64
	stepSize      float64
	timeList      []float64
	outputBufSize uint64
	totalNumCols  uint64
	numBlocks     uint64
	blockNames    []string
	blockNameMap  map[string]uint64
	blockInfo     []blockData
}

// CountData is a container holding the data corresponding to a reaction data
// output block consisting of a number of columns
type CountData struct {
	Col       [][]float64
	DataTypes []uint16
}

// blockData stores metadata related to a given data block such as the
// data name, number of data columns, the type of data stored (int/double),
// and the internal offset into the buffer at which the data can be found.
type blockData struct {
	name      string
	numCols   uint64
	dataTypes []uint16
	offset    uint64
}

// BlockNames returns the list of available blocknames
func (d *MCellData) BlockNames() []string {
	return d.blockNames
}

// IDtoBlockName returns the blockname corresponding to the given id
func (d *MCellData) IDtoBlockName(id uint64) (string, error) {
	if id < 0 || id >= uint64(len(d.blockNames)) {
		return "", fmt.Errorf("requested id is out of range")
	}
	return d.blockNames[id], nil
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

// OutputTimes returns a slice with the output times corresponding to the
// column data (either computed from STEP or via ITERATION_LIST/TIME_LIST)
// NOTE: In the case of STEP we cache the timelist after the first request
func (d *MCellData) OutputTimes() []float64 {
	if d.OutputType() == Step && len(d.timeList) == 0 {
		d.timeList = make([]float64, d.BlockSize())
		for i := uint64(0); i < d.BlockSize(); i++ {
			d.timeList[i] = d.StepSize() * float64(i)
		}
	}
	return d.timeList
}

// BlockDataByName returns the data stored in the data block of the given name
// as a CountData struct
func (d *MCellData) BlockDataByName(name string) (*CountData, error) {
	id, ok := d.blockNameMap[name]
	if !ok {
		return nil, fmt.Errorf("dataset %s not found", name)
	}

	return d.BlockDataByID(id)
}

// BlockDataByID returns the data stored in the data block of the given ID
// as a CountData struct
func (d *MCellData) BlockDataByID(id uint64) (*CountData, error) {
	if id < 0 || id >= d.numBlocks {
		return nil, fmt.Errorf("supplied data ID %d is out of range", id)
	}

	entry := d.blockInfo[id]
	output := &CountData{}
	output.Col = make([][]float64, entry.numCols)
	for i := uint64(0); i < entry.numCols; i++ {
		output.Col[i] = make([]float64, 0, d.blockSize)
		output.DataTypes = append(output.DataTypes, entry.dataTypes[i])
	}

	row := uint64(0)
	stream := uint64(0)
	bufLoc := uint64(0)
	// read all rows until we hit the total blockSize
	for row < d.blockSize {

		// forward to the next stream block if we're done parsing the current one
		if row > stream*d.outputBufSize {
			offset := d.outputBufSize
			// last partial block of length <= d.outputBufSize requires special treatment
			if row-d.blockSize < d.outputBufSize {
				offset = d.blockSize - row
			}
			// forward to beginning of stream block
			bufLoc = stream * d.outputBufSize * d.totalNumCols * len_double

			// forward to location within stream block
			bufLoc += offset * entry.offset * len_double

			stream++
		}

		// read current row
		for i := uint64(0); i < entry.numCols; i++ {
			loc := (d.buffer)[bufLoc:]
			val := loc.float64NoSlice() //d.buffer[bufLoc:].float64NoSlice()
			output.Col[i] = append(output.Col[i], val)
			bufLoc += len_double
		}
		row++
	}

	return output, nil
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

	var totalCols uint64
	for count := uint64(0); count < data.numBlocks; count++ {

		e := blockData{}
		e.offset = totalCols
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
		totalCols += e.numCols
		for c := uint64(0); c < e.numCols; c++ {
			e.dataTypes = append(e.dataTypes, data.buffer.uint16())
		}
		data.blockInfo = append(data.blockInfo, e)
	}
	data.totalNumCols = totalCols

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

func (b *readBuf) float64NoSlice() float64 {
	v := math.Float64frombits(binary.LittleEndian.Uint64(*b))
	return v
}
