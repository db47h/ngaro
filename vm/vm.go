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

// Package vm blah.
// TODO: add
// complete file i/o
//	- a reset func: clear stacks/reset ip to 0, accept Options (input / output may need to be reset as well)
//	- a disasm func
//	- discard the RuneWriter interface. Implement it by hand.
//	- mod=ve options out of the Options struct, and remove that struct.
package vm

import (
	"bufio"
	"os"
)

// Cell is the raw type stored in a memory location.
type Cell int32

type opcode Cell

// ngaro Virtual Machine Opcodes.
const (
	OpNop opcode = iota
	OpLit
	OpDup
	OpDrop
	OpSwap
	OpPush
	OpPop
	OpLoop
	OpJump
	OpReturn
	OpGtJump
	OpLtJump
	OpNeJump
	OpEqJump
	OpFetch
	OpStore
	OpAdd
	OpSub
	OpMul
	OpDimod
	OpAnd
	OpOr
	OpXor
	OpShl
	OpShr
	OpZeroExit
	OpInc
	OpDec
	OpIn
	OpOut
	OpWait
)

const (
	portCount   = 64
	dataSize    = 1024
	addressSize = 1024
)

// RuneReader is the interface that wraps the ReadRune method.
//
// ReadRune reads a single UTF-8 encoded Unicode character
// and returns the rune and its size in bytes. If no character is
// available, err will be set.
type RuneReader interface {
	ReadRune() (r rune, size int, err error)
}

// RuneWriter is the interface that wraps the WriteRune method.
//
// WriteRune writes a single Unicode code point, returning
// the number of bytes written and any error.
type RuneWriter interface {
	WriteRune(r rune) (size int, err error)
}

// Option interface
type Option interface {
	set(p *Instance)
}

type optionFunc func(p *Instance)

func (f optionFunc) set(p *Instance) {
	f(p)
}

// Options for VM instance creation.
var Options = struct {
	DataSize    func(size int) Option     // Sets the data stack size.
	AddressSize func(size int) Option     // Sets the address stack size.
	Input       func(r RuneReader) Option // Sets the input RuneReader.
	Output      func(r RuneWriter) Option // Sets the output writer.
}{
	DataSize: func(size int) Option {
		var f optionFunc = func(i *Instance) { i.data = make([]Cell, size) }
		return f
	},
	AddressSize: func(size int) Option {
		var f optionFunc = func(i *Instance) { i.address = make([]Cell, size) }
		return f
	},
	Input: func(r RuneReader) Option {
		var f optionFunc = func(i *Instance) { i.PushInput(r) }
		return f
	},
	Output: func(r RuneWriter) Option {
		var f optionFunc = func(i *Instance) { i.output = r }
		return f
	},
}

// Instance represents an ngaro VM instance
type Instance struct {
	ip        int
	sp        int
	rsp       int
	Image     Image
	data      []Cell
	address   []Cell
	ports     []Cell
	imageFile string
	input     RuneReader
	output    RuneWriter
	insCount  int64
}

// New creates a new ProcessingUnit
func New(image Image, imageFile string, opts ...Option) *Instance {
	i := &Instance{
		ip:        0,
		sp:        -1,
		rsp:       -1,
		Image:     image,
		ports:     make([]Cell, portCount),
		imageFile: imageFile,
	}
	for _, opt := range opts {
		opt.set(i)
	}
	if i.data == nil {
		i.data = make([]Cell, 1024)
	}
	if i.address == nil {
		i.address = make([]Cell, 1024)
	}
	if i.input == nil {
		i.input = bufio.NewReader(os.Stdin)
	}
	if i.output == nil {
		i.output = bufio.NewWriter(os.Stdout)
	}
	return i
}

// Data returns the data stack. Note that value changes will be reflected in the
// instance's stack, but reslicing will not affect it. To add/remove values on
// the data stack, use the Push and Pop functions.
func (i *Instance) Data() []Cell {
	return i.data[:i.sp+1]
}

// Address returns the address stack. Note that value changes will be reflected
// in the instance's stack, but reslicing will not affect it. To add/remove
// values on the address stack, use the Rpush and Rpop functions.
func (i *Instance) Address() []Cell {
	return i.address[:i.rsp+1]
}

// InstructionCount returns the number of instructions executed so far.
func (i *Instance) InstructionCount() int64 {
	return i.insCount
}
