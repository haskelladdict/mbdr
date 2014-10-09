package libmbd

import (
	"bytes"
	"compress/bzip2"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
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

// BlockDataByRegex returns a map with all datasets whose name matched the
// supplied regex. The map keys are the dataset names, the values are the
// corresponding count data items.
func (d *MCellData) BlockDataByRegex(selection string) (map[string]*CountData, error) {

	regex, err := regexp.Compile(selection)
	if err != nil {
		return nil, err
	}

	outputData := make(map[string]*CountData)
	names := d.BlockNames()
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
	stream := uint64(1)
	bufLoc := d.outputBufSize * len_double * entry.offset
	// read all rows until we hit the total blockSize
	for row < d.blockSize {

		// forward to the next stream block if we're done parsing the current one
		if row >= stream*d.outputBufSize {
			offset := d.outputBufSize
			// last partial block of length <= d.outputBufSize requires special treatment
			if d.blockSize-row < d.outputBufSize {
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

// ReadHeader opens the binary mcell data file and parses the header without
// reading the actual data. This provides efficient access to metadata and
// the names of stored data blocks. After calling this function the buffer
// field of MCellData is set to nil since no data is parsed.
func ReadHeader(filename string) (*MCellData, error) {
	fileRaw, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fileRaw.Close()
	file := bzip2.NewReader(fileRaw)

	var data MCellData
	err = parseHeader(file, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

// Read header opens the binary mcell data file and parses the header and the
// actual data stored. If only access to the metadata is required, it is much
// more efficient to only call ReadHeader directly.
func Read(filename string) (*MCellData, error) {
	fileRaw, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fileRaw.Close()
	file := bzip2.NewReader(fileRaw)

	var data MCellData
	err = parseHeader(file, &data)
	if err != nil {
		return nil, err
	}

	err = parseData(file, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

// parseData read all of the binary count data into MCellData's properly
// preallocated []byte buffer
func parseData(r io.Reader, data *MCellData) error {

	// compute required capacity of buffer
	// NOTE: we allocate an additional data.blockSize to avoid re-allocation
	var err error
	capacity := data.blockSize*data.totalNumCols*len_double + data.blockSize
	data.buffer, err = readAll(r, int64(capacity))
	if err != nil {
		return err
	}
	return nil
}

// checkAPITag reads the API tag inside the data set and makes sure it
// matches the required version
func checkAPITag(r io.Reader) error {
	receivedAPITag := make([]byte, len(requiredAPITag))
	if _, err := io.ReadFull(r, receivedAPITag); err != nil {
		return err
	}
	if string(receivedAPITag) != requiredAPITag {
		return fmt.Errorf("incorrect API Tag %s", receivedAPITag)
	}
	return nil
}

// parseBlockInfo reads the pertinent data block information such as the
// time step, time list, number of data blocks etc.
func parseBlockInfo(r io.Reader, data *MCellData) error {

	var err error
	if data.outputType, err = readUint16(r); err != nil {
		return err
	}

	if data.blockSize, err = readUint64(r); err != nil {
		return err
	}

	var length uint64
	if length, err = readUint64(r); err != nil {
		return err
	}

	switch data.outputType {
	case Step:
		if data.stepSize, err = readFloat64(r); err != nil {
			return err
		}

	case TimeList:
		var time float64
		for i := uint64(0); i < length; i++ {
			if time, err = readFloat64(r); err != nil {
				return err
			}
			data.timeList = append(data.timeList, time)
		}

	case IterationList:
		var iter float64
		for i := uint64(0); i < length; i++ {
			if iter, err = readFloat64(r); err != nil {
				return err
			}
			data.timeList = append(data.timeList, iter)
		}

	default:
		return fmt.Errorf("encountered unknown data output type")
	}

	if data.outputBufSize, err = readUint64(r); err != nil {
		return err
	}

	if data.numBlocks, err = readUint64(r); err != nil {
		return err
	}

	return nil
}

// parseBlockNames extract the names of data blocks contained with the data file
func parseBlockNames(r io.Reader, data *MCellData) error {

	// initialize blockname map
	data.blockNameMap = make(map[string]uint64)

	var totalCols uint64
	var err error
	buf := []byte{0}
	for count := uint64(0); count < data.numBlocks; count++ {

		e := blockData{}
		e.offset = totalCols

		// find end of next block name
		var nameBuf bytes.Buffer
		for {
			if _, err = io.ReadFull(r, buf); err != nil {
				return err
			}
			if string(buf) == "\x00" {
				break
			}
			nameBuf.Write(buf)
		}

		e.name = nameBuf.String()
		data.blockNames = append(data.blockNames, e.name)
		data.blockNameMap[e.name] = count

		if e.numCols, err = readUint64(r); err != nil {
			return err
		}
		totalCols += e.numCols
		for c := uint64(0); c < e.numCols; c++ {
			dataType, err := readUint16(r)
			if err != nil {
				return err
			}
			e.dataTypes = append(e.dataTypes, dataType)
		}
		data.blockInfo = append(data.blockInfo, e)
	}
	data.totalNumCols = totalCols

	return nil
}

// parseHeader reads the header of the binary mcell data file to check the API
// version and retrieve general information regarding the data contained
// within (number of datablocks, block names, ...).
func parseHeader(r io.Reader, data *MCellData) error {

	if err := checkAPITag(r); err != nil {
		return err
	}

	// skip next byte - this is a defect in the mcell binary output format
	dummy := make([]byte, 1)
	if _, err := io.ReadFull(r, dummy); err != nil {
		return err
	}

	if err := parseBlockInfo(r, data); err != nil {
		return err
	}

	if err := parseBlockNames(r, data); err != nil {
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
	//	*b = (*b)[2:]
	return v
}

func (b *readBuf) uint64() uint64 {
	v := binary.LittleEndian.Uint64(*b)
	//	*b = (*b)[8:]
	return v
}

func (b *readBuf) float64() float64 {
	v := math.Float64frombits(binary.LittleEndian.Uint64(*b))
	//	*b = (*b)[8:]
	return v
}

func (b *readBuf) float64NoSlice() float64 {
	v := math.Float64frombits(binary.LittleEndian.Uint64(*b))
	return v
}

// readUint16 reads an uint16 from a io.Reader
func readUint16(r io.Reader) (uint16, error) {
	buf := make(readBuf, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return buf.uint16(), nil
}

// readUint64 reads an uint64 from an io.Reader
func readUint64(r io.Reader) (uint64, error) {
	buf := make(readBuf, 8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return buf.uint64(), nil
}

// readFloat64 reads a float64 from an io.Reader
func readFloat64(r io.Reader) (float64, error) {
	buf := make(readBuf, 8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return buf.float64(), nil
}

// readAll is taken verbatim from ioutil in the standard library and we use
// it to read the binary count data into a preallocated buffer of the correct
// size. Using a correctly preallocated buffer is critical especially for large
// binary data files in the multi GB range to avoid excessive memory use due
// to uncollected memory
func readAll(r io.Reader, capacity int64) (b []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, capacity))
	// If the buffer overflows, we will get bytes.ErrTooLarge.
	// Return that as an error. Any other panic remains.
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			panic(e)
		}
	}()
	_, err = buf.ReadFrom(r)
	return buf.Bytes(), err
}
