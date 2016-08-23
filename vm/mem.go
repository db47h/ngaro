// This file is part of ngaro - https://github.com/db47h/ngaro
//
// Copyright 2016 Denis Bernard <db047h@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vm

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"unsafe"

	"github.com/pkg/errors"
)

// load32 loads a 32 bits image.
func load32(mem []Cell, r io.Reader, fileCells int) error {
	var b = make([]byte, 4)
	var p int
	for p < len(mem) {
		_, err := io.ReadFull(r, b)
		if err != nil {
			if err != io.EOF {
				return errors.Wrap(err, "cell read failed")
			}
			break
		}
		mem[p] = Cell(int32(binary.LittleEndian.Uint32(b)))
		p++
	}
	if p != fileCells {
		return errors.Errorf("read %c cells, expected %d", p, fileCells)
	}
	return nil
}

// load64 loads a 64 bits image.
func load64(mem []Cell, r io.Reader, fileCells int) error {
	var b = make([]byte, 8)
	var p int
	for p < len(mem) {
		_, err := io.ReadFull(r, b)
		if err != nil {
			if err != io.EOF {
				return errors.Wrap(err, "cell read failed")
			}
			break
		}
		v := int64(binary.LittleEndian.Uint64(b))
		n := Cell(v)
		if int64(n) != v {
			return errors.Errorf("64 bits value %d at memory location %d too large", v, p)
		}
		mem[p] = n
		p++
	}
	if p != fileCells {
		return errors.Errorf("read %c cells, expected %d", p, fileCells)
	}
	return nil
}

// Load loads a memory image from file fileName. Returns a VM Cell slice ready
// to run from, the actual number of cells read from the file and any error. The
// cellBits parameter specifies the number of bits per Cell in the file.
//
// The returned slice should have its length equal to the maximum of the
// requested minimum size and the image file size + 1024 free cells.
func Load(fileName string, minSize, cellBits int) (mem []Cell, fileCells int, err error) {
	switch cellBits {
	case 0:
		cellBits = int(unsafe.Sizeof(Cell(0))) * 8
	case 32, 64:
	default:
		return nil, 0, errors.Errorf("loading of %d bits images is not supported", cellBits)
	}
	f, err := os.Open(fileName)
	if err != nil {
		return nil, 0, errors.Wrap(err, "open failed")
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, 0, errors.Wrap(err, "fstat failed")
	}
	sz := st.Size()
	if sz > int64((^uint(0))>>1) { // MaxInt
		return nil, 0, errors.Errorf("%v: file too large", fileName)
	}
	fileCells = int(sz / int64(cellBits/8))
	// make sure there are at least 1024 free cells at the end of the image
	imgCells := fileCells + 1024
	if minSize > imgCells {
		imgCells = minSize
	}
	mem = make([]Cell, imgCells)
	switch cellBits {
	case 32:
		err = load32(mem, bufio.NewReader(f), fileCells)
	case 64:
		err = load64(mem, bufio.NewReader(f), fileCells)
	}
	if err != nil {
		return nil, fileCells, errors.Wrap(err, "load failed")
	}
	return mem, fileCells, nil
}

// Save saves a Cell slice to an memory image file. The cellBits parameter
// specifies the number of bits per Cell in the file.
func Save(fileName string, mem []Cell, cellBits int) error {
	f, err := os.Create(fileName)
	if err != nil {
		return errors.Wrap(err, "create failed")
	}
	w := bufio.NewWriter(f)
	defer func() {
		w.Flush()
		f.Close()
		// delete file on error
		if err != nil {
			os.Remove(fileName)
		}
	}()
	if cellBits == 0 {
		cellBits = int(unsafe.Sizeof(Cell(0))) * 8
	}
	switch cellBits {
	case 32:
		var b [4]byte
		for k, v := range mem {
			nv := int32(v)
			if Cell(nv) != v {
				return errors.Errorf("64 bits value %d at memory location %d too large", v, k)
			}
			binary.LittleEndian.PutUint32(b[:], uint32(nv))
			if _, err = w.Write(b[:]); err != nil {
				return errors.Wrap(err, "write failed")
			}
		}
	case 64:
		var b [8]byte
		for _, v := range mem {
			binary.LittleEndian.PutUint64(b[:], uint64(v))
			if _, err = w.Write(b[:]); err != nil {
				return errors.Wrap(err, "write failed")
			}
		}
	default:
		return errors.Errorf("saving to %d bits images is not supported", cellBits)
	}
	return errors.Wrap(err, "save failed")
}

// DecodeString returns the string starting at position start in the specified
// slice. Strings stored in the slice must be zero terminated. The trailing '\0'
// is not returned.
func DecodeString(mem []Cell, start Cell) string {
	pos := int(start)
	end := pos
	for ; end < len(mem) && mem[end] != 0; end++ {
	}
	str := make([]byte, end-pos)
	for idx, c := range mem[pos:end] {
		str[idx] = byte(c)
	}
	return string(str)
}

// EncodeString writes the given string at position start in specified slice
// and terminates it with a '\0' Cell.
func EncodeString(mem []Cell, start Cell, s string) {
	pos := int(start)
	for _, c := range []byte(s) {
		mem[pos] = Cell(c)
		pos++
	}
	mem[pos] = 0
}
