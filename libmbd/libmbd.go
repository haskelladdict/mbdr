// libmbd contains data structures and helper routines for extracting reaction
// data from binary mcell reaction data output files
package libmbd

import (
	"fmt"
	"github.com/haskelladdict/mbdr/parser/util"
	"regexp"
)

// MCellData tracks the data contained in the binary mcell file as well as
// relevant metadata to retrieve specific data items.
// NOTE: Depending on the API version of the binary output data not all fields
// are defined
type MCellData struct {
	Buffer        util.ReadBuf
	API           int
	OutputType    uint16
	BlockSize     uint64
	StepSize      float64
	TimeList      []float64
	OutputBufSize uint64 // only for API >= 2
	TotalNumCols  uint64 // only for API >= 2
	NumBlocks     uint64
	BlockNames    []string
	BlockNameMap  map[string]uint64
	BlockInfo     []BlockData // only for API >= 2
}

// CountData is a container holding the data corresponding to a reaction data
// output block consisting of a number of columns
type CountData struct {
	Col       [][]float64
	DataTypes []uint16
}

// BlockData stores metadata related to a given data block such as the
// data name, number of data columns, the type of data stored (int/double),
// and the internal offset into the buffer at which the data can be found.
type BlockData struct {
	Name      string
	NumCols   uint64
	DataTypes []uint16
	Offset    uint64
}

// convenience consts describing the length of certain C types in bytes
const (
	LenUint16_t = 2
	LenUint64_t = 8
	LenDouble   = 8
)

// enumeration describing the output type of the included data
const (
	Step uint16 = 1 << iota
	TimeListType
	IterationListType
)

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

// DateType returns the output type (STEP, ITERATION_LIST/TIME_LIST)
func (d *MCellData) DataType() uint16 {
	return d.OutputType
}

// OutputStepLen returns the output step length. NOTE: The returns value is only
// meaningful is outputType == Step, otherwise this function returns 0
func (d *MCellData) OutputStepLen() float64 {
	return d.StepSize
}

// OutputTimes returns a slice with the output times corresponding to the
// column data (either computed from STEP or via ITERATION_LIST/TIME_LIST)
// NOTE: In the case of STEP we cache the timelist after the first request
func (d *MCellData) OutputTimes() []float64 {
	if d.DataType() == Step && len(d.TimeList) == 0 {
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
func (d *MCellData) BlockDataByID(id uint64) (*CountData, error) {
	if id < 0 || id >= d.NumBlocks {
		return nil, fmt.Errorf("supplied data ID %d is out of range", id)
	}

	entry := d.BlockInfo[id]
	output := &CountData{}
	output.Col = make([][]float64, entry.NumCols)
	for i := uint64(0); i < entry.NumCols; i++ {
		output.Col[i] = make([]float64, 0, d.BlockSize)
		output.DataTypes = append(output.DataTypes, entry.DataTypes[i])
	}

	row := uint64(0)
	stream := uint64(1)
	bufLoc := d.OutputBufSize * LenDouble * entry.Offset
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
			bufLoc = stream * d.OutputBufSize * d.TotalNumCols * LenDouble

			// forward to location within stream block
			bufLoc += offset * entry.Offset * LenDouble

			stream++
		}

		// read current row
		for i := uint64(0); i < entry.NumCols; i++ {
			loc := (d.Buffer)[bufLoc:]
			val := loc.Float64NoSlice() //d.buffer[bufLoc:].float64NoSlice()
			output.Col[i] = append(output.Col[i], val)
			bufLoc += LenDouble
		}
		row++
	}

	return output, nil
}
