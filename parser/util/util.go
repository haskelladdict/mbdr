// util contains low lever helper function for parsing byte buffers
package util

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

// convenience consts describing the length of certain C types in bytes
const (
	LenByte    = 1
	LenUint16  = 2
	LenUint32  = 4
	LenUint64  = 8
	LenFloat64 = 8
)

// readBuf and helper function convert between a byte slice and an underlying
// integer type
// NOTE: This code was take almost verbatim from archive/zip/reader from the
// standard library
type ReadBuf []byte

func (b *ReadBuf) uint16() uint16 {
	v := binary.LittleEndian.Uint16(*b)
	return v
}

func (b *ReadBuf) uint32() uint32 {
	v := binary.LittleEndian.Uint32(*b)
	return v
}

func (b *ReadBuf) uint64() uint64 {
	v := binary.LittleEndian.Uint64(*b)
	return v
}

// float conversions
func (b *ReadBuf) float64() float64 {
	v := math.Float64frombits(binary.LittleEndian.Uint64(*b))
	return v
}

func (b *ReadBuf) Float64NoSlice() float64 {
	v := math.Float64frombits(binary.LittleEndian.Uint64(*b))
	return v
}

// readByte reads a single byte from an io.Reader
func ReadByte(r io.Reader) (byte, error) {
	buf := []byte{0}
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return buf[0], nil
}

// readUint16 reads an uint16 from an io.Reader
func ReadUint16(r io.Reader) (uint16, error) {
	buf := make(ReadBuf, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return buf.uint16(), nil
}

// readUint32 reads an uint32 from an io.Reader
func ReadUint32(r io.Reader) (uint32, error) {
	buf := make(ReadBuf, 4)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return buf.uint32(), nil
}

// readUint64 reads an uint64 from an io.Reader
func ReadUint64(r io.Reader) (uint64, error) {
	buf := make(ReadBuf, 8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return buf.uint64(), nil
}

// readFloat64 reads a float64 from an io.Reader
func ReadFloat64(r io.Reader) (float64, error) {
	buf := make(ReadBuf, 8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return buf.float64(), nil
}

// ReadAll is taken verbatim from ioutil in the standard library and we use
// it to read the binary count data into a preallocated buffer of the correct
// size. Using a correctly preallocated buffer is critical especially for large
// binary data files in the multi GB range to avoid excessive memory use due
// to uncollected memory
func ReadAll(r io.Reader, capacity int64) (b []byte, err error) {
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
