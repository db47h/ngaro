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
	"strconv"
)

// Cell is the raw type stored in a memory location.
type Cell int32

const (
	portCount   = 1024
	dataSize    = 1024
	addressSize = 1024
)

// Instance represents an Ngaro VM instance.
type Instance struct {
	PC        int    // Program Counter (aka. Instruction Pointer)
	Image     Image  // Memory image
	Ports     []Cell // I/O ports
	fileCells int
	sp        int
	rsp       int
	data      []Cell
	address   []Cell
	insCount  int64
	inH       map[Cell]InHandler
	outH      map[Cell]OutHandler
	waitH     map[Cell]WaitHandler
	imageFile string
	shrink    bool
	input     io.Reader
	output    Terminal
	fid       Cell
	files     map[Cell]*os.File
}

// Option interface
type Option func(*Instance) error

// DataSize sets the data stack size. It will not erase the stack, but data nay
// be lost if set to a smaller size. The default is 1024 cells.
func DataSize(size int) Option {
	return func(i *Instance) error {
		if size <= len(i.data) {
			i.data = i.data[:size]
		} else {
			t := make([]Cell, size)
			copy(t, i.data[:i.sp+1])
		}
		return nil
	}
}

// AddressSize sets the address stack size. It will not erase the stack, but data nay
// be lost if set to a smaller size. The default is 1024 cells.
func AddressSize(size int) Option {
	return func(i *Instance) error {
		if size <= len(i.address) {
			i.data = i.address[:size]
		} else {
			t := make([]Cell, size)
			copy(t, i.address[:i.rsp+1])
		}
		return nil
	}
}

// Input pushes the given RuneReader on top of the input stack.
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

// Shrink enables or disables image shrinking when saving it. The default is
// false.
func Shrink(shrink bool) Option {
	return func(i *Instance) error { i.shrink = shrink; return nil }
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
// The image parameter is the Cell array used as memory by the VM. Usually
// loaded from file with the Load function. Note that New expects the lenght of
// the slice to be the actual image file size (in Cells) and its capacity set to
// the run-time image size, so New will expand the slice to its full capacity
// before using it.
//
// The imageFile parameter is the fileName that will be used to dump the
// contents of the memory image. It does not have to exist or even be writable
// as long as no user program requests an image dump.
//
// Options will be set by calling SetOptions.
func New(image Image, imageFile string, opts ...Option) (*Instance, error) {
	i := &Instance{
		PC:        0,
		sp:        -1,
		rsp:       -1,
		Image:     image[:cap(image)],
		Ports:     make([]Cell, portCount),
		inH:       make(map[Cell]InHandler),
		outH:      make(map[Cell]OutHandler),
		waitH:     make(map[Cell]WaitHandler),
		imageFile: imageFile,
		files:     make(map[Cell]*os.File),
		fid:       1,
		fileCells: len(image),
	}

	// default Wait Handlers
	for _, p := range []Cell{1, 2, 4, 5, 8} {
		i.waitH[p] = (*Instance).Wait
	}

	if err := i.SetOptions(opts...); err != nil {
		return nil, err
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
// instance's stack, but re-slicing will not affect it. To add/remove values on
// the data stack, use the Push and Pop functions.
func (i *Instance) Data() []Cell {
	if i.sp < len(i.data) {
		return i.data[:i.sp+1]
	}
	return i.data
}

// Address returns the address stack. Note that value changes will be reflected
// in the instance's stack, but re-slicing will not affect it. To add/remove
// values on the address stack, use the Rpush and Rpop functions.
func (i *Instance) Address() []Cell {
	if i.rsp < len(i.address) {
		return i.address[:i.rsp+1]
	}
	return i.address
}

// InstructionCount returns the number of instructions executed so far.
func (i *Instance) InstructionCount() int64 {
	return i.insCount
}

type errWriter struct {
	w   io.Writer
	err error
}

func (w *errWriter) Write(p []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err = w.w.Write(p)
	if err != nil {
		w.err = err
	}
	return n, err
}

func (w *errWriter) dumpSlice(a []Cell) error {
	l := len(a) - 1
	if l >= 0 {
		for i := 0; i < l; i++ {
			io.WriteString(w, strconv.Itoa(int(a[i])))
			w.Write([]byte{' '})
		}
		io.WriteString(w, strconv.Itoa(int(a[l])))
	}
	return w.err
}

// Dump dumps the virtual machine stacks and image to the specified io.Writer.
func (i *Instance) Dump(w io.Writer) error {
	ew := &errWriter{w: w}
	ew.Write([]byte{'\x1C'})
	ew.dumpSlice(i.data[:i.sp+1])
	ew.Write([]byte{'\x1D'})
	ew.dumpSlice(i.address[:i.rsp+1])
	ew.Write([]byte{'\x1D'})
	return ew.dumpSlice(i.Image[:i.fileCells])
}
