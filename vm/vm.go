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
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
)

const (
	portCount   = 1024
	dataSize    = 1024
	addressSize = 1024
)

// Bits per Cell
const (
	// Compute the size of a Cell
	_m       = ^uCell(0)
	_log     = _m>>8&1 + _m>>16&1 + (_m>>31)>>1&1 // >>31>>1 is to trick go vet
	CellBits = (1 << _log) << 3
)

// Instance represents an Ngaro VM instance.
type Instance struct {
	PC          int    // Program Counter (aka. Instruction Pointer)
	Mem         []Cell // Memory image
	Ports       []Cell // I/O ports
	tos         Cell   // cell on top of stack
	sp          int
	rsp         int
	rtos        Cell
	data        []Cell
	address     []Cell
	insCount    int64
	inH         map[Cell]InHandler
	outH        map[Cell]OutHandler
	waitH       map[Cell]WaitHandler
	sEnc        Codec
	opHandler   OpcodeHandler
	imageFile   string
	input       io.Reader
	output      Terminal
	fid         Cell
	files       map[Cell]*os.File
	memDump     func(string, []Cell) error
	clockPeriod time.Duration
}

// Option interface
type Option func(*Instance) error

// ClockPeriod sets the period between VM ticks. A zero or negative period means no pause.
// Default is zero.
func ClockPeriod(period time.Duration) Option {
	return func(i *Instance) error {
		i.clockPeriod = period
		return nil
	}
}

// DataSize sets the data stack size. It will not erase the stack, and will
// panic if the requested size is not sufficient to hold the current stack. The
// default is 1024 cells.
func DataSize(size int) Option {
	return func(i *Instance) error {
		if size < i.sp {
			return errors.Errorf("requested stack size too small to hold current stack: %d < %d", size, i.sp)
		}
		size++
		if size <= len(i.data) {
			i.data = i.data[:size]
		} else {
			i.data = make([]Cell, size)
			copy(i.address, i.data[:i.sp])
		}
		return nil
	}
}

// AddressSize sets the address stack size. It will not erase the stack, and will
// panic if the requested size is not sufficient to hold the current stack. The
// default is 1024 cells.
func AddressSize(size int) Option {
	return func(i *Instance) error {
		if size < i.rsp {
			return errors.Errorf("requested stack size too small to hold current stack: %d < %d", size, i.rsp)
		}
		size++
		if size <= len(i.address) {
			i.address = i.address[:size]
		} else {
			i.address = make([]Cell, size)
			copy(i.address, i.address[:i.rsp])
		}
		return nil
	}
}

// Input pushes the given io.Reader on top of the input stack.
func Input(r io.Reader) Option {
	return func(i *Instance) error { i.PushInput(r); return nil }
}

// Output configures the output Terminal. For simple I/O, the helper function
// NewVT100Terminal will build a Terminal wrapper around an io.Writer.
func Output(t Terminal) Option {
	return func(i *Instance) error {
		i.output = t
		return nil
	}
}

// SaveMemImage overrides the memory image dump function called when writing 1 to I/O port 4.
// The default is to call:
//
//	Save(i.imageFile, i.Mem, 0)
//
// This is to allow saving images of different Cell sizes and to enable
// implementations of specific languages (like Retro) to do image shrinking
// based on some value in the VM instance's memory.
func SaveMemImage(fn func(filename string, mem []Cell) error) Option {
	return func(i *Instance) error { i.memDump = fn; return nil }
}

// InHandler is the function prototype for custom IN handlers.
type InHandler func(i *Instance, port Cell) error

// OutHandler is the function prototype for custom OUT handlers.
type OutHandler func(i *Instance, v, port Cell) error

// WaitHandler is the function prototype for custom WAIT handlers.
type WaitHandler func(i *Instance, v, port Cell) error

// BindInHandler binds the porvided IN handler to the given port.
//
// The default IN handler behaves according to the specification: it reads the
// corresponding port value from Ports[port] and pushes it to the data stack.
// After reading, the value of Ports[port] is reset to 0.
//
// Custom hamdlers do not strictly need to interract with Ports field. It is
// however recommended that they behave the same as the default.
func BindInHandler(port Cell, handler InHandler) Option {
	return func(i *Instance) error {
		i.inH[port] = handler
		return nil
	}
}

// BindOutHandler binds the porvided OUT handler to the given port.
//
// The default OUT handler just stores the given value in Ports[port].
// A common use of OutHandler when using buffered I/O is to flush the output
// writer when anything is written to port 3. Such handler just ignores the
// written value, leaving Ports[3] as is.
func BindOutHandler(port Cell, handler OutHandler) Option {
	return func(i *Instance) error {
		i.outH[port] = handler
		return nil
	}
}

// BindWaitHandler binds the porvided WAIT handler to the given port.
//
// WAIT handlers are called only if the value the following conditions are both
// true:
//
//  - the value of the bound I/O port is not 0
//  - the value of I/O port 0 is not 1
//
// Upon completion, a WAIT handler should call the WaitReply method which will
// set the value of the bound port and set the value of port 0 to 1.
func BindWaitHandler(port Cell, handler WaitHandler) Option {
	return func(i *Instance) error {
		i.waitH[port] = handler
		return nil
	}
}

// OpcodeHandler is the prototype for opcode handler functions. When an opcode
// handler is called, the VM's PC points to the opcode. Opcode handlers must take
// care of updating the VM's PC.
type OpcodeHandler func(i *Instance, opcode Cell) error

// BindOpcodeHandler binds the given function to handle custom opcodes (i.e.
// opcodes with a negative value).
//
// When an opcode handler is called, the VM's PC points to the opcode. Opcode
// handlers must take care of updating the VM's PC.
func BindOpcodeHandler(handler OpcodeHandler) Option {
	return func(i *Instance) error {
		i.opHandler = handler
		return nil
	}
}

// StringCodec delegates string encoding/decoding in the memory image to the
// specified Codec. This is needed in file I/O where filenames are read from
// memory. Clients that make use of these I/O calls must configure a
// StringCodec. For Retro style encoding (one byte per Cell, 0 terminated),
// retro.StringCodec can be used as Codec. Implementations using othe encoding
// schemes, must provide their own Codec.
func StringCodec(e Codec) Option {
	return func(i *Instance) error {
		i.sEnc = e
		return nil
	}
}

// SetOptions sets the provided options.
func (i *Instance) SetOptions(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(i); err != nil {
			return err
		}
	}
	return nil
}

// New creates a new Ngaro Virtual Machine instance.
//
// The mem parameter is the Cell array used as memory image by the VM.
//
// The imageFile parameter is the fileName that will be used to dump the
// contents of the memory image. It does not have to exist or even be writable
// as long as no user program requests an image dump.
//
// Options will be set by calling SetOptions.
func New(mem []Cell, imageFile string, opts ...Option) (*Instance, error) {
	i := &Instance{
		PC:        0,
		Mem:       mem,
		Ports:     make([]Cell, portCount),
		inH:       make(map[Cell]InHandler),
		outH:      make(map[Cell]OutHandler),
		waitH:     make(map[Cell]WaitHandler),
		imageFile: imageFile,
		files:     make(map[Cell]*os.File),
		fid:       1,
		memDump:   func(filename string, mem []Cell) error { return Save(filename, mem, 0) },
	}

	// default Wait Handlers
	for _, p := range []Cell{1, 2, 4, 5, 8} {
		i.waitH[p] = (*Instance).Wait
	}

	if err := i.SetOptions(opts...); err != nil {
		return nil, errors.Wrap(err, "SetOptions failed")
	}
	if i.data == nil {
		i.SetOptions(DataSize(dataSize))
	}
	if i.address == nil {
		i.SetOptions(AddressSize(addressSize))
	}
	return i, nil
}

// Data returns the data stack. Note that value changes will be reflected in the
// instance's stack, but re-slicing will not affect it. To add/remove values on
// the data stack, use the Push and Pop functions.
func (i *Instance) Data() []Cell {
	if i.sp < 1 {
		return nil
	}
	return append(i.data[2:i.sp+1], i.tos)
}

// Address returns the address stack. Note that value changes will be reflected
// in the instance's stack, but re-slicing will not affect it. To add/remove
// values on the address stack, use the Rpush and Rpop functions.
func (i *Instance) Address() []Cell {
	if i.rsp < 1 {
		return nil
	}
	return append(i.address[2:i.rsp+1], i.rtos)
}

// InstructionCount returns the number of instructions executed so far.
func (i *Instance) InstructionCount() int64 {
	return i.insCount
}
