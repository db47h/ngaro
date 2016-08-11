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
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"unsafe"
)

// Image encapsulates a VM's memory
type Image []Cell

// Load loads an image from file fileName. The returned slice should have its
// length equal to the number of cells in the file and its capacity equal to the
// maximum of the requested capacity and the image file size + 1024 free cells.
// When using this slice to create a new VM, New will get the length to track
// the image file size and expand the slice to its full capacity.
func Load(fileName string, capacity int) (Image, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	sz := st.Size()
	if sz > int64((^uint(0))>>1) { // MaxInt
		return nil, fmt.Errorf("Load %v: file too large", fileName)
	}
	fileCells := int(sz / int64(unsafe.Sizeof(Cell(0))))
	// make sure there are at least 1024 free cells at the end of the image
	imgCells := fileCells + 1024
	if capacity > imgCells {
		imgCells = capacity
	}
	i := make(Image, fileCells, imgCells)
	err = binary.Read(f, binary.LittleEndian, i[:fileCells])
	if err != nil {
		return nil, err
	}
	return i, nil
}

// Save saves the image. If the shrink parameter is true, only the portion of
// the image from offset 0 to HERE will be saved.
func (i Image) Save(fileName string, shrink bool) error {
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	if shrink {
		i = i[0:i[3]]
	}
	return binary.Write(f, binary.LittleEndian, i)
}

// DecodeString returns the string starting at position start in the image.
// Strings stored in the image must be zero terminated. The trailing '\0' is
// not returned.
func (i Image) DecodeString(start Cell) string {
	pos := int(start)
	end := pos
	for ; end < len(i) && i[end] != 0; end++ {
	}
	str := make([]byte, end-pos)
	for idx, c := range i[pos:end] {
		str[idx] = byte(c)
	}
	return string(str)
}

// EncodeString writes the given string at position start in the Image and
// terminates it with a '\0' Cell.
func (i Image) EncodeString(start Cell, s string) {
	pos := int(start)
	for _, c := range []byte(s) {
		i[pos] = Cell(c)
		pos++
	}
	i[pos] = 0
}

// Disassemble disassembles the cells at position pc and returns the position of
// the next valid opcode and the disassembly string.
func (i Image) Disassemble(pc int) (next int, disasm string) {
	var d bytes.Buffer
	op := i[pc]
	if op >= Cell(len(opcodes)) {
		d.WriteString("call ")
		d.WriteString(strconv.Itoa(int(op)))
	} else {
		d.WriteString(opcodes[op])
	}
	pc++
	switch op {
	case OpLoop, OpJump, OpGtJump, OpLtJump, OpNeJump, OpEqJump:
		if pc < len(i) {
			d.WriteByte(' ')
		}
		fallthrough
	case OpLit:
		if pc < len(i) {
			d.WriteString(strconv.Itoa(int(i[pc])))
			return pc + 1, d.String()
		}
		d.WriteString("???")
	}
	return pc, d.String()
}
