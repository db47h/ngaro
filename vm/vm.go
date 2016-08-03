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
// TODO:
//	- port i/o hooks
//	- complete file i/o
//	- add a reset func: clear stacks/reset ip to 0, accept Options (input / output may need to be reset as well)
//	- add a disasm func
//	- implement communication with host go program via channels (in io)
//	- go routines that leverage channels (watch out for the panic handler, we should have a global `done` channel)
//	- BUG: I/O trashes ports in interactive mode. For example, the following returns 0 instead of the image size:
//		-1 5 out 0 0 out wait 5 in putn
package vm

import "io"

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

// Option interface
type Option func(*Instance) error

// DataSize sets the data stack size.
func DataSize(size int) Option {
	return func(i *Instance) error { i.data = make([]Cell, size); return nil }
}

// AddressSize sets the address stack size.
func AddressSize(size int) Option {
	return func(i *Instance) error { i.address = make([]Cell, size); return nil }
}

// Input pushes the given RuneReader on top of the input stack.
func Input(r io.Reader) Option {
	return func(i *Instance) error { i.PushInput(r); return nil }
}

// Output sets the output Writer.
func Output(w io.Writer) Option {
	return func(i *Instance) error { i.output = newWriter(w); return nil }
}

// Shrink enables or disables image shrinking when saving it.
func Shrink(shrink bool) Option {
	return func(i *Instance) error { i.shrink = shrink; return nil }
}

// Instance represents an ngaro VM instance.
type Instance struct {
	PC        int
	sp        int
	rsp       int
	Image     Image
	data      []Cell
	address   []Cell
	ports     []Cell
	imageFile string
	shrink    bool
	input     io.RuneReader
	output    runeWriter
	insCount  int64
}

// New creates a new Ngaro Virtual Machine instance.
func New(image Image, imageFile string, opts ...Option) (*Instance, error) {
	i := &Instance{
		PC:        0,
		sp:        -1,
		rsp:       -1,
		Image:     image,
		ports:     make([]Cell, portCount),
		imageFile: imageFile,
	}
	for _, opt := range opts {
		if err := opt(i); err != nil {
			return nil, err
		}
	}
	if i.data == nil {
		i.data = make([]Cell, 1024)
	}
	if i.address == nil {
		i.address = make([]Cell, 1024)
	}
	return i, nil
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
