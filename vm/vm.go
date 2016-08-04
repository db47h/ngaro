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

// Package vm implements the Ngaro VM.
// TODO:
//	- complete file i/o
//	- add a reset func: clear stacks/reset ip to 0, accept Options (input / output may need to be reset as well)
//	- add a disassembly function.
//	- go routines
//	- BUG: I/O trashes ports in interactive mode. For example, the following returns 0 instead of the image size:
//		-1 5 out 0 0 out wait 5 in putn
package vm

import (
	"errors"
	"io"
)

// Cell is the raw type stored in a memory location.
type Cell int32

const (
	portCount   = 1024
	dataSize    = 1024
	addressSize = 1024
)

// Option interface
type Option func(*Instance) error

// DataSize sets the data stack size. It will not erase the stack, but data nay
// be lost if set to a smaller size.
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
// be lost if set to a smaller size.
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

// Output sets the output Writer. If the isatty flag is set to true, the output
// will be treated as a raw terminal and special handling of some control
// characters will apply. This will also enable the extended terminal support.
func Output(w io.Writer, isatty bool) Option {
	return func(i *Instance) error {
		i.tty = isatty
		i.output = newWriter(w)
		return nil
	}
}

// Shrink enables or disables image shrinking when saving it.
func Shrink(shrink bool) Option {
	return func(i *Instance) error { i.shrink = shrink; return nil }
}

// ErrUnhandled is a sentinel error for WAIT handlers. See WaitHandler.
var ErrUnhandled = errors.New("Unknown port value")

// IOCallback is a port IN/OUT handler function.
type IOCallback func(old Cell) (new Cell, err error)

// InHandler will make any IN on the given port call the provided hadler.
// The actual port value will be passed to the handler and the handler's return
// value will be pushed onto the stack. An example no-op handler:
//
//	func handleIn(v vm.Cell) (vm.Cell, error) {
//		return v
//	}
func InHandler(port Cell, handler IOCallback) Option {
	return func(i *Instance) error {
		i.inH[int(port)] = handler
		return nil
	}
}

// OutHandler will make any OUT on the given port call the provided handler.
// The OUT value will be passed to the handler and the handler's return value
// will be written to the port.
func OutHandler(port Cell, handler IOCallback) Option {
	return func(i *Instance) error {
		i.outH[int(port)] = handler
		return nil
	}
}

// WaitHandler will make any WAIT on the given port call the provided handler.
// The port value will be passed to the handler and the handler's return value
// will be written to the port.
//
// The handler will only be called if port 0 value is 0 and if the bound port
// value is != 0. Wait handlers can be used to override default port behavior.
// If the returned error is ErrUnhandled, the returned value and error will be
// ignored, and the default implementation will handle the WAIT.
func WaitHandler(port int, handler IOCallback) Option {
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

// Instance represents an Ngaro VM instance.
type Instance struct {
	PC        int
	sp        int
	rsp       int
	Image     Image
	data      []Cell
	address   []Cell
	ports     []Cell
	insCount  int64
	inH       map[int]IOCallback
	outH      map[int]IOCallback
	waitH     map[int]IOCallback
	imageFile string
	shrink    bool
	input     io.RuneReader
	output    runeWriter
	tty       bool
}

// New creates a new Ngaro Virtual Machine instance.
func New(image Image, imageFile string, opts ...Option) (*Instance, error) {
	i := &Instance{
		PC:        0,
		sp:        -1,
		rsp:       -1,
		Image:     image,
		ports:     make([]Cell, portCount),
		inH:       make(map[int]IOCallback, portCount),
		outH:      make(map[int]IOCallback, portCount),
		waitH:     make(map[int]IOCallback, portCount),
		imageFile: imageFile,
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
