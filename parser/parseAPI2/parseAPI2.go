// Package parseAPI2 contains the infrastructure for parsing MCell binary
// reaction data output that was written according to API version
// MCELL_BINARY_API_2.
package parseAPI2

import (
	"bytes"
	"fmt"
	"io"

	"github.com/haskelladdict/mbdr/libmbd"
	"github.com/haskelladdict/mbdr/parser/util"
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
	var err error
	capacity := data.BlockSize*data.TotalNumCols*util.LenFloat64 + data.BlockSize
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
	if data.OutputListType, err = util.ReadUint16(r); err != nil {
		return err
	}

	if data.BlockSize, err = util.ReadUint64(r); err != nil {
		return err
	}

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

	if data.OutputBufSize, err = util.ReadUint64(r); err != nil {
		return err
	}

	if data.NumBlocks, err = util.ReadUint64(r); err != nil {
		return err
	}

	return nil
}

// parseBlockNames extract the names of data blocks contained with the data file
func parseBlockNames(r io.Reader, data *libmbd.MCellData) error {

	// initialize blockname map
	data.BlockNameMap = make(map[string]uint64)

	var totalCols uint64
	var err error
	buf := []byte{0}
	for count := uint64(0); count < data.NumBlocks; count++ {

		e := libmbd.BlockData{}
		e.Offset = totalCols

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

		e.Name = nameBuf.String()
		data.BlockNames = append(data.BlockNames, e.Name)
		data.BlockNameMap[e.Name] = count

		if e.NumCols, err = util.ReadUint64(r); err != nil {
			return err
		}
		totalCols += e.NumCols
		for c := uint64(0); c < e.NumCols; c++ {
			dataType, err := util.ReadUint16(r)
			if err != nil {
				return err
			}
			e.DataTypes = append(e.DataTypes, dataType)
		}
		data.BlockInfo = append(data.BlockInfo, e)
	}
	data.TotalNumCols = totalCols

	return nil
}

// parseHeader reads the header of the binary mcell data file to check the API
// version and retrieve general information regarding the data contained
// within (number of datablocks, block names, ...).
func parseHeader(r io.Reader, data *libmbd.MCellData) error {

	// skip first byte - this is a defect in the mcell binary output format
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
