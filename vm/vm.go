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
	"io"
	"os"
)

// Cell is the raw type stored in a memory location.
type Cell int32

// UCell is the unsigned counterpart of Cell
type UCell uint32

type opcode Cell

// Opcodes
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

type runeWriter interface {
	WriteRune(r rune) (size int, err error)
	Flush() error
}

// Option interface
type Option interface {
	set(p *Instance)
}

type optionFunc func(p *Instance)

func (f optionFunc) set(p *Instance) {
	f(p)
}

// Options for VM
var Options = struct {
	DataSize    func(size int) Option
	AddressSize func(size int) Option
}{
	DataSize: func(size int) Option {
		var f optionFunc = func(p *Instance) { p.data = make([]Cell, size) }
		return f
	},
	AddressSize: func(size int) Option {
		var f optionFunc = func(p *Instance) { p.address = make([]Cell, size) }
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
	input     io.RuneReader
	output    runeWriter
}

// New creates a new ProcessingUnit
func New(image Image, imageFile string, opts ...Option) *Instance {
	p := &Instance{
		ip:        0,
		sp:        -1,
		rsp:       -1,
		Image:     image,
		ports:     make([]Cell, portCount),
		imageFile: imageFile,
		input:     bufio.NewReader(os.Stdin),
		output:    bufio.NewWriter(os.Stdout),
	}
	for _, opt := range opts {
		opt.set(p)
	}
	if p.data == nil {
		p.data = make([]Cell, 1024)
	}
	if p.address == nil {
		p.address = make([]Cell, 1024)
	}
	return p
}

// // Data returns a copy of the data stack for inspection
// func (p *Instance) Data() []Cell {
// 	if p.sp < 0 {
// 		return nil
// 	}
// 	r := make([]Cell, p.sp+1)
// 	copy(r, p.data)
// 	return r
// }
//
// // Address returns a copy of the address stack for inspection
// func (p *Instance) Address() []Cell {
// 	if p.rsp < 0 {
// 		return nil
// 	}
// 	r := make([]Cell, p.rsp+1)
// 	copy(r, p.address)
// 	return r
// }
