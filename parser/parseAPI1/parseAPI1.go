// parseAPI1 contains the infrastructure for parsing MCell binary reaction data
// output that was written according to API version MCELL_BINARY_API_1.
package parseAPI1

import (
	"bytes"
	"fmt"
	"github.com/haskelladdict/mbdr/libmbd"
	"github.com/haskelladdict/mbdr/parser/util"
	"io"
)

// Header parses the header without reading the actual data. This provides
// efficient access to metadata and the names of stored data blocks. After calling
// this function the buffer field of MCellData is set to nil since no data is parsed.
func Header(r io.Reader, data *libmbd.MCellData) (*libmbd.MCellData, error) {
	if err := parseHeader(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

// Data reads all of the binary count data into MCellData's properly
// preallocated []byte buffer
func Data(r io.Reader, data *libmbd.MCellData) (*libmbd.MCellData, error) {
	// compute required capacity of buffer
	// NOTE: we allocate an additional data.blockSize to avoid re-allocation
	var capacity uint64
	for i := uint64(0); i < data.NumBlocks; i++ {
		var itemLen uint64
		switch data.BlockEntries[i].Type {
		case 0:
			itemLen = util.LenUint32

		case 1:
			itemLen = util.LenFloat64

		default:
			return nil, fmt.Errorf("encountered incorrect data type %d", data.BlockEntries[i].Type)
		}
		capacity += data.BlockSize * itemLen
	}
	capacity += data.BlockSize

	var err error
	data.Buffer, err = util.ReadAll(r, int64(capacity))
	if err != nil {
		return nil, err
	}
	return data, nil
}

// parseBlockInfo reads the pertinent data block information such as the
// time step, time list, number of data blocks etc.
func parseBlockInfo(r io.Reader, data *libmbd.MCellData) error {

	var err error
	var outputType uint32
	if outputType, err = util.ReadUint32(r); err != nil {
		return err
	}
	// NOTE: since output type is 0, 1, or 2 this conversion is safe
	data.OutputListType = uint16(outputType) + 1

	var length uint64
	if length, err = util.ReadUint64(r); err != nil {
		return err
	}

	switch data.OutputListType {
	case libmbd.Step:
		if data.StepSize, err = util.ReadFloat64(r); err != nil {
			return err
		}

	case libmbd.TimeListType:
		var time float64
		for i := uint64(0); i < length; i++ {
			if time, err = util.ReadFloat64(r); err != nil {
				return err
			}
			data.TimeList = append(data.TimeList, time)
		}

	case libmbd.IterationListType:
		var iter float64
		for i := uint64(0); i < length; i++ {
			if iter, err = util.ReadFloat64(r); err != nil {
				return err
			}
			data.TimeList = append(data.TimeList, iter)
		}

	default:
		return fmt.Errorf("encountered unknown data output type")
	}

	return nil
}

// parseBlockNames extract the names of data blocks contained with the data file
func parseBlockNames(r io.Reader, data *libmbd.MCellData) error {

	// initialize blockname map
	data.BlockNameMap = make(map[string]uint64)

	// parse block names
	buf := []byte{0}
	for count := uint64(0); count < data.NumBlocks; count++ {
		var nameBuf bytes.Buffer
		for {
			if _, err := io.ReadFull(r, buf); err != nil {
				return err
			}
			if string(buf) == "\x00" {
				break
			}
			nameBuf.Write(buf)
		}

		name := nameBuf.String()
		data.BlockNames = append(data.BlockNames, name)
		data.BlockNameMap[name] = count
	}

	return nil
}

// parseHeader reads the header of the binary mcell data file to check the API
// version and retrieve general information regarding the data contained
// within (number of datablocks, block names, ...).
func parseHeader(r io.Reader, data *libmbd.MCellData) error {

	// skip first byte - this is a defect in the mcell binary output format
	dummy := []byte{0}
	if _, err := io.ReadFull(r, dummy); err != nil {
		return err
	}

	var err error
	if data.BlockSize, err = util.ReadUint64(r); err != nil {
		return err
	}

	var numBlocks uint32
	if numBlocks, err = util.ReadUint32(r); err != nil {
		return err
	}
	data.NumBlocks = uint64(numBlocks)

	if err := parseBlockNames(r, data); err != nil {
		return nil
	}

	if err := parseBlockInfo(r, data); err != nil {
		return nil
	}

	for i := uint64(0); i < data.NumBlocks; i++ {
		var entry libmbd.BlockEntry
		if entry.Type, err = util.ReadByte(r); err != nil {
			return err
		}
		if entry.Start, err = util.ReadUint64(r); err != nil {
			return err
		}
		if entry.End, err = util.ReadUint64(r); err != nil {
			return err
		}
		data.BlockEntries = append(data.BlockEntries, entry)
	}

	if data.NumBlocks != 0 {
		data.Offset = data.BlockEntries[0].Start
	}

	return nil
}
