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
	"io"
	"strconv"

	"github.com/db47h/ngaro/vm"
)

var opcodes = [...]string{
	"nop",
	"lit",
	"dup",
	"drop",
	"swap",
	"push",
	"pop",
	"loop",
	"jump",
	";",
	">jump",
	"<jump",
	"!jump",
	"=jump",
	"@",
	"!",
	"+",
	"-",
	"*",
	"/mod",
	"and",
	"or",
	"xor",
	"<<",
	">>",
	"0;",
	"1+",
	"1-",
	"in",
	"out",
	"wait",
}

var opcodeIndex = make(map[string]vm.Cell)

func init() {
	for i, v := range opcodes {
		opcodeIndex[v] = vm.Cell(i)
	}
}

// Assemble compiles assembly read from the supplied io.Reader and returns the
// resulting image and error if any.
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

// Disassemble disassembles the cells in the given slice at position pc to the
// specified io.Writer and returns the position of the next valid opcode.
func Disassemble(i []vm.Cell, pc int, w io.Writer) (next int) {
	op := i[pc]
	if op < 0 || op >= vm.Cell(len(opcodes)) {
		io.WriteString(w, "call ")
		io.WriteString(w, strconv.Itoa(int(op)))
	} else if op != vm.OpLit {
		io.WriteString(w, opcodes[op])
	}
	pc++
	switch op {
	case vm.OpLoop, vm.OpJump, vm.OpGtJump, vm.OpLtJump, vm.OpNeJump, vm.OpEqJump:
		if pc < len(i) {
			w.Write([]byte{' '})
		}
		fallthrough
	case vm.OpLit:
		if pc < len(i) {
			io.WriteString(w, strconv.Itoa(int(i[pc])))
			return pc + 1
		}
		io.WriteString(w, "???")
	}
	return pc
}
