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

package asm

import (
	"fmt"
	"io"
	"strconv"

	"github.com/db47h/ngaro/vm"
)

var opcodes = [...][]string{
	{"nop"},
	{"lit"},
	{"dup"},
	{"drop"},
	{"swap"},
	{"push"},
	{"pop"},
	{"loop"},
	{"jump", "jmp"},
	{";", "ret"},
	{">jump", "jgt"},
	{"<jump", "jlt"},
	{"!jump", "jne"},
	{"=jump", "jeq"},
	{"@"},
	{"!"},
	{"+", "add"},
	{"-", "sub"},
	{"*", "mul"},
	{"/mod"},
	{"and"},
	{"or"},
	{"xor"},
	{"<<", "shl"},
	{">>", "asr"},
	{"0;", "0ret"},
	{"1+", "inc"},
	{"1-", "dec"},
	{"in"},
	{"out"},
	{"wait"},
}

// Assemble compiles assembly read from the supplied io.Reader and returns the
// resulting memory image and error if any.
//
// Then name parameter is used only in error messages to name the source of the
// error. If the io.Reader is a file, name should be the file name.
//
// The returned error, if not nil, can safely be cast to an ErrAsm value that
// will contain up to 10 entries.
func Assemble(name string, r io.Reader) (img []vm.Cell, err error) {
	p := newParser()
	img, err = p.Parse(name, r)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// Disassemble writes a disassembly of the cells in the given slice at position
// pc to the specified io.Writer and returns the position of the next valid
// opcode and any write error.
//
// Note that some instructions will Disassemble like:
//
//	.dat 860	( call 860 )
//
// This is because the cell value is 860 and  disassenbler cannot determine if
// it's an implicit call or raw data. Disassembling it this way reminds you that
// this could be a call, while allowing the output to be passed as-is to the
// assembler.
func Disassemble(i []vm.Cell, pc int, w io.Writer) (next int, err error) {
	op := i[pc]
	b := make([]byte, 0, 40)
	if op < 0 || op >= vm.Cell(len(opcodes)) {
		b = append(b, ".dat "...)
		b = strconv.AppendInt(b, int64(int(op)), 10)
		b = append(b, "\t( call "...)
		b = strconv.AppendInt(b, int64(int(op)), 10)
		b = append(b, ' ', ')')
	} else if op != vm.OpLit {
		b = append(b, opcodes[op][0]...)
	}
	pc++
	switch op {
	case vm.OpLoop, vm.OpJump, vm.OpGtJump, vm.OpLtJump, vm.OpNeJump, vm.OpEqJump:
		if pc < len(i) {
			b = append(b, ' ')
		}
		fallthrough
	case vm.OpLit:
		if pc < len(i) {
			b = strconv.AppendInt(b, int64(int(i[pc])), 10)
			_, err = w.Write(b)
			return pc + 1, err
		}
		b = append(b, "???"...)
	}
	_, err = w.Write(b)
	return pc, err
}

// DisassembleAll writes a disassembly of all cells in the given slice to
// the specified io.Writer. The base argument specifies the real address of the
// frist cell (i[0]). It will return any write error.
func DisassembleAll(i []vm.Cell, base int, w io.Writer) error {
	for pc := 0; pc < len(i); {
		_, err := fmt.Fprintf(w, "% 10d\t", base+pc)
		if err != nil {
			return err
		}
		pc, _ = Disassemble(i, pc, w)
		_, err = w.Write([]byte{'\n'})
		if err != nil {
			return err
		}
	}
	return nil
}
