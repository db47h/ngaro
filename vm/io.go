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
	"unicode/utf8"
	"unsafe"
)

// Terminal encapsulates methods provided by a terminal output.
//
// WriteRune writes a single Unicode code point, returning the number of bytes written and any error.
//
// Flush writes any buffered unwritten output.
//
// Size returns the width and height of the terminal window.
//
// Clear clears the terminal window.
//
// CursorPos moves the cursor at he given position.
//
// FgColor and BgColor respectively set the foreground and background color.
type Terminal interface {
	io.Writer
	WriteRune(r rune) (size int, err error)
	Flush() error
	Size() (width int, height int)
	Clear()
	CursorPos(x, y int)
	FgColor(fg int)
	BgColor(bg int)
	Port8Enabled() bool
}

// PushInput sets r as the current input RuneReader for the VM. When this reader
// reaches EOF, the previously pushed reader will be used.
func (i *Instance) PushInput(r io.Reader) {
	// dont use a multi reader unless necessary
	switch in := i.input.(type) {
	case nil: // no input yet, single assign
		i.input = newRuneReader(r)
	case *multiRuneReader:
		in.pushReader(r)
	default:
		i.input = &multiRuneReader{[]io.RuneReader{newRuneReader(r), i.input}}
	}
}

// In is the default IN handler for all ports.
func (i *Instance) In(port Cell) error {
	i.Push(i.Ports[port])
	i.Ports[port] = 0
	return nil
}

// Out is the default OUT handler for all ports.
func (i *Instance) Out(v, port Cell) error {
	if port == 3 {
		if i.output == nil {
			return nil
		}
		return i.output.Flush()
	}
	i.Ports[port] = v
	return nil
}

// WaitReply writes the value v to the given port and sets port 0 to 1. This
// should only be used by WAIT port handlers.
func (i *Instance) WaitReply(v, port Cell) {
	i.Ports[port] = v
	i.Ports[0] = 1
}

// Wait is the default WAIT handler bound to ports 1, 2, 4, 5 and 8. It can be
// called manually by custom handlers that override default behaviour.
func (i *Instance) Wait(v, port Cell) error {
	if v == 0 {
		return nil
	}

	switch port {
	case 1: // input
		if v == 1 {
			if i.input == nil {
				return io.EOF
			}
			r, size, err := i.input.ReadRune()
			if size > 0 {
				i.WaitReply(Cell(r), 1)
			} else {
				i.WaitReply(utf8.RuneError, 1)
				if err != nil {
					return err
				}
			}
		}
	case 2: // output
		if v == 1 {
			r := rune(i.Pop())
			if i.output != nil {
				var err error
				if r < 0 {
					i.output.Clear()
				} else {
					_, err = i.output.WriteRune(r)
				}
				if err != nil {
					return err
				}
			}
			i.WaitReply(0, 2)
		}
	case 4: // FileIO
		if v != 0 {
			i.Ports[0] = 1
			switch v {
			case 1: // save image
				i.Image.Save(i.imageFile, i.shrink)
				i.Ports[4] = 0
			case 2: // include file
				i.Ports[4] = 0
				f, err := os.Open(i.Image.DecodeString(i.Pop()))
				if err != nil {
					return err
				}
				i.PushInput(f)
			default:
				i.Ports[4] = 0
			}
		}
	case 5: // VM capabilities
		if i.Ports[5] != 0 {
			switch i.Ports[5] {
			case -1:
				// image size
				i.Ports[5] = Cell(len(i.Image))
			// -2, -3, -4: canvas related
			case -5:
				// data depth
				i.Ports[5] = Cell(i.sp + 1)
			case -6:
				// address depth
				i.Ports[5] = Cell(i.rsp + 1)
			// -7: mouse enabled
			case -8:
				// unix time
				i.Ports[5] = Cell(time.Now().Unix())
			case -9:
				// exit VM
				i.Ports[5] = 0
				i.PC = len(i.Image) - 1 // will be incremented when returning
			case -10:
				// environment query
				src, dst := i.data[i.sp], i.data[i.sp-1]
				i.sp -= 2
				i.Image.EncodeString(dst, os.Getenv(i.Image.DecodeString(src)))
				i.Ports[5] = 0
			case -11:
				// console width
				if i.output != nil {
					w, _ := i.output.Size()
					i.Ports[5] = Cell(w)
				} else {
					i.Ports[5] = 0
				}
			case -12:
				// console height
				if i.output != nil {
					_, h := i.output.Size()
					i.Ports[5] = Cell(h)
				} else {
					i.Ports[5] = 0
				}
			case -13:
				i.Ports[5] = Cell(unsafe.Sizeof(Cell(0)) * 8)
			// -14: endianness
			case -15:
				// port 8 enabled
				if i.output != nil && i.output.Port8Enabled() {
					i.Ports[5] = -1
				} else {
					i.Ports[5] = 0
				}
			case -16:
				i.Ports[5] = Cell(len(i.data))
			case -17:
				i.Ports[5] = Cell(len(i.address))
			default:
				i.Ports[5] = 0
			}
			i.Ports[0] = 1
		}
	case 8:
		if v := i.Ports[8]; v != 0 && i.output != nil {
			switch i.Ports[8] {
			case 1:
				i.output.CursorPos(int(i.data[i.sp-1]), int(i.data[i.sp]))
				i.sp -= 2
			case 2:
				i.output.FgColor(int(i.Pop()))
			case 3:
				i.output.BgColor(int(i.Pop()))
			}
			i.WaitReply(0, 8)
		}
	}
	return nil
}
