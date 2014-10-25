// Package libmbd contains data structures and helper routines for extracting reaction
// data from binary mcell reaction data output files
package libmbd

import (
	"fmt"
	"regexp"

	"github.com/haskelladdict/mbdr/parser/util"
)

// enumeration describing the output type of the included data
const (
	Step uint16 = 1 << iota
	TimeListType
	IterationListType
)

// list of currently know API versions
const (
	API1 = "MCELL_BINARY_API_1"
	API2 = "MCELL_BINARY_API_2"
)

// MCellData tracks the data contained in the binary mcell file as well as
// relevant metadata to retrieve specific data items.
// NOTE: Depending on the API version of the binary output data not all fields
// are defined
type MCellData struct {
	Buffer         util.ReadBuf
	OutputListType uint16
	BlockSize      uint64
	StepSize       float64
	TimeList       []float64
	NumBlocks      uint64
	BlockNames     []string
	BlockNameMap   map[string]uint64
	API            string
	API1Data
	API2Data
}

// API1Data are data items specific to API version 1 of the mcell binary output
// format.
type API1Data struct {
	Offset       uint64 // offset into data buffer
	BlockEntries []BlockEntry
}

// BlockEntry is used for API version 1. It stores the beginning and end of
// each data block within the data buffer.
type BlockEntry struct {
	Type  byte
	Start uint64
	End   uint64
}

// API2Data are data items specific to API version 1 of the mcell binary output
// format.
type API2Data struct {
	OutputBufSize uint64
	TotalNumCols  uint64
	BlockInfo     []BlockData
}

// BlockData is used for API version 2. It stores metadata for a given data block
// such as the data name, number of data columns, the type of data
// stored (int/double), and the internal offset into the buffer at which the
// data can be found.
type BlockData struct {
	Name      string
	NumCols   uint64
	DataTypes []uint16
	Offset    uint64
}

// CountData is a container holding the data corresponding to a reaction data
// output block consisting of a number of columns
type CountData struct {
	Col       [][]float64
	DataTypes []uint16
}

// DataNames returns the list of available blocknames
func (d *MCellData) DataNames() []string {
	return d.BlockNames
}

// IDtoBlockName returns the blockname corresponding to the given id
func (d *MCellData) IDtoBlockName(id uint64) (string, error) {
	if id < 0 || id >= uint64(len(d.BlockNames)) {
		return "", fmt.Errorf("requested id is out of range")
	}
	return d.BlockNames[id], nil
}

// NumDataBlocks returns the number of available datablocks
func (d *MCellData) NumDataBlocks() uint64 {
	return d.NumBlocks
}

// BlockLen returns the number of output iterations per datablock
func (d *MCellData) BlockLen() uint64 {
	return d.BlockSize
}

// OutputType returns the output type (STEP, ITERATION_LIST/TIME_LIST)
func (d *MCellData) OutputType() uint16 {
	return d.OutputListType
}

// OutputStepLen returns the output step length. NOTE: The returns value is only
// meaningful is OutputListType == Step, otherwise this function returns 0
func (d *MCellData) OutputStepLen() float64 {
	return d.StepSize
}

// OutputTimes returns a slice with the output times corresponding to the
// column data (either computed from STEP or via ITERATION_LIST/TIME_LIST)
// NOTE: In the case of STEP we cache the timelist after the first request
func (d *MCellData) OutputTimes() []float64 {
	if d.OutputType() == Step && len(d.TimeList) == 0 {
		d.TimeList = make([]float64, d.BlockLen())
		for i := uint64(0); i < d.BlockLen(); i++ {
			d.TimeList[i] = d.OutputStepLen() * float64(i)
		}
	}
	return d.TimeList
}

// BlockDataByRegex returns a map with all datasets whose name matched the
// supplied regex. The map keys are the dataset names, the values are the
// corresponding count data items.
func (d *MCellData) BlockDataByRegex(selection string) (map[string]*CountData, error) {

	regex, err := regexp.Compile(selection)
	if err != nil {
		return nil, err
	}

	outputData := make(map[string]*CountData)
	names := d.DataNames()
	for _, n := range names {
		if regex.MatchString(n) {
			countData, err := d.BlockDataByName(n)
			if err != nil {
				return nil, err
			}
			outputData[n] = countData
		}
	}
	return outputData, nil
}

// BlockDataByName returns the data stored in the data block of the given name
// as a CountData struct
func (d *MCellData) BlockDataByName(name string) (*CountData, error) {
	id, ok := d.BlockNameMap[name]
	if !ok {
		return nil, fmt.Errorf("dataset %s not found", name)
	}

	return d.BlockDataByID(id)
}

// BlockDataByID returns the data stored in the data block of the given ID
// as a CountData struct
// NOTE: This is the only method of MCellData which is API sensitive
func (d *MCellData) BlockDataByID(id uint64) (*CountData, error) {
	if id < 0 || id >= d.NumBlocks {
		return nil, fmt.Errorf("supplied data ID %d is out of range", id)
	}

	var c *CountData
	var e error
	switch d.API {
	case API1:
		c, e = d.blockDataAPI1(id)
	case API2:
		c, e = d.blockDataAPI2(id)
	default:
		c = nil
		e = fmt.Errorf("unknown API type %s in BlockDataByID\n", d.API)
	}
	return c, e
}

// blockDataAPI1 returns count data for mcell binary API version 2. It returns
// the data stored in the data block of the given ID as a CountData struct
func (d *MCellData) blockDataAPI1(id uint64) (*CountData, error) {

	entry := d.BlockEntries[id]
	output := &CountData{}
	output.Col = make([][]float64, 1)
	output.Col[0] = make([]float64, 0, d.BlockSize)
	output.DataTypes = append(output.DataTypes, uint16(entry.Type))

	loc := entry.Start - d.Offset
	var buf util.ReadBuf
	switch entry.Type {
	case 0:
		var val uint32
		for i := uint64(0); i < d.BlockSize; i++ {
			buf = (d.Buffer)[loc:]
			val = buf.Uint32()
			output.Col[0] = append(output.Col[0], float64(val))
			loc += util.LenUint32
		}
		// sanity check
		if loc != entry.End-d.Offset {
			return nil, fmt.Errorf("did not properly reach end of data block %d\n", id)
		}

	case 1:
		var val float64
		for i := uint64(0); i < d.BlockSize; i++ {
			buf = (d.Buffer)[loc:]
			val = buf.Float64()
			output.Col[0] = append(output.Col[0], val)
			loc += util.LenFloat64
		}
		// sanity check
		if loc != entry.End-d.Offset {
			return nil, fmt.Errorf("did not properly reach end of data block %d\n", id)
		}
	}
	return output, nil
}

// blockDataAPI2 returns count data for mcell binary API version 2. It returns
// the data stored in the data block of the given ID as a CountData struct
func (d *MCellData) blockDataAPI2(id uint64) (*CountData, error) {

	entry := d.BlockInfo[id]
	output := &CountData{}
	output.Col = make([][]float64, entry.NumCols)
	for i := uint64(0); i < entry.NumCols; i++ {
		output.Col[i] = make([]float64, 0, d.BlockSize)
		output.DataTypes = append(output.DataTypes, entry.DataTypes[i])
	}

	row := uint64(0)
	stream := uint64(1)
	var loc uint64
	// NOTE: We need this to be able to deal with checkpoint files for which
	// the total number of items may be smaller than the output buffer size
	if d.BlockSize < d.OutputBufSize {
		loc = d.BlockSize * util.LenFloat64 * entry.Offset
	} else {
		loc = d.OutputBufSize * util.LenFloat64 * entry.Offset
	}
	// read all rows until we hit the total blockSize
	for row < d.BlockSize {

		// forward to the next stream block if we're done parsing the current one
		if row >= stream*d.OutputBufSize {
			offset := d.OutputBufSize
			// last partial block of length <= d.outputBufSize requires special treatment
			if d.BlockSize-row < d.OutputBufSize {
				offset = d.BlockSize - row
			}
			// forward to beginning of stream block
			loc = stream * d.OutputBufSize * d.TotalNumCols * util.LenFloat64

			// forward to location within stream block
			loc += offset * entry.Offset * util.LenFloat64

			stream++
		}
		// read current row
		for i := uint64(0); i < entry.NumCols; i++ {
			buf := (d.Buffer)[loc:]
			val := buf.Float64() //d.buffer[bufLoc:].float64NoSlice()
			output.Col[i] = append(output.Col[i], val)
			loc += util.LenFloat64
		}
		row++
	}

	return output, nil
}
