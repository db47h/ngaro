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
	"encoding/binary"
	"fmt"
	"os"
	"unsafe"
)

// Load loads a memory image from file fileName. Returns a VM Cell slice ready to run
// from, the actual number of cells read from the file and any error.
//
// The returned slice should have its length equal to the maximum of the
// requested minimum size and the image file size + 1024 free cells.
func Load(fileName string, minSize int) (mem []Cell, fileCells int, err error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	sz := st.Size()
	if sz > int64((^uint(0))>>1) { // MaxInt
		return nil, 0, fmt.Errorf("Load %v: file too large", fileName)
	}
	fileCells = int(sz / int64(unsafe.Sizeof(Cell(0))))
	// make sure there are at least 1024 free cells at the end of the image
	imgCells := fileCells + 1024
	if minSize > imgCells {
		imgCells = minSize
	}
	mem = make([]Cell, imgCells)
	err = binary.Read(f, binary.LittleEndian, mem[:fileCells])
	if err != nil {
		return nil, 0, err
	}
	return mem, fileCells, nil
}

// Save saves a Cell slice to an memory image file.
func Save(mem []Cell, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	return binary.Write(f, binary.LittleEndian, mem)
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
