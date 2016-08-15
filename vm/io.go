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
	"unsafe"
)

// Terminal encapsulates methods provided by a terminal output. Apart from
// WriteRune, all methods can be implemented as no-ops if the underlying output
// does not support the corresponding functionality.
//
// WriteRune writes a single Unicode code point, returning the number of bytes written and any error.
//
// Flush writes any buffered unwritten output.
//
// Size returns the width and height of the terminal window. Should return 0, 0
// if unsupported.
//
// Clear clears the terminal window and moves the cursor to the top left.
//
// MoveCursor moves the cursor to the specified column and row.
//
// FgColor and BgColor respectively set the foreground and background color of
// all characters subsequently written.
//
// Port8Enabled should return true if the MoveCursor, FgColor and BgColor
// methods have any effect.
type Terminal interface {
	io.Writer
	Flush() error
	Size() (width int, height int)
	Clear()
	MoveCursor(x, y int)
	FgColor(fg int)
	BgColor(bg int)
	Port8Enabled() bool
}

func (i *Instance) openfile(name string, mode Cell) Cell {
	var flags int
	switch mode {
	case 0:
		flags = os.O_RDONLY
	case 1:
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	case 2:
		flags = os.O_RDWR | os.O_CREATE | os.O_APPEND
	case 3:
		flags = os.O_RDWR
	default:
		return 0
	}
	f, err := os.OpenFile(name, flags, 0666)
	if err != nil {
		return 0
	}
	for ; i.files[i.fid] != nil; i.fid++ {
	}
	i.files[i.fid] = f
	return i.fid
}

// PushInput sets r as the current input RuneReader for the VM. When this reader
// reaches EOF, the previously pushed reader will be used.
func (i *Instance) PushInput(r io.Reader) {
	// dont use a multi reader unless necessary
	switch in := i.input.(type) {
	case nil:
		// no input yet, single assign
		i.input = r
	case *multiReader:
		// stack it
		in.pushReader(r)
	default:
		// build multireader from two single readers
		i.input = &multiReader{[]io.Reader{r, i.input}}
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
			var b [1]byte
			if i.input == nil {
				return io.EOF
			}
			size, err := i.input.Read(b[:])
			if size > 0 {
				i.WaitReply(Cell(b[0]), 1)
			} else {
				i.WaitReply(-1, 1)
				if err != nil {
					return err
				}
			}
		}
	case 2: // output
		if v == 1 {
			c := i.Pop()
			if i.output != nil {
				var err error
				if c < 0 {
					i.output.Clear()
				} else {
					_, err = i.output.Write([]byte{byte(c)})
				}
				if err != nil {
					return err
				}
			}
			i.WaitReply(0, 2)
		}
	case 4: // FileIO
		if v != 0 {
			var b [1]byte
			switch v {
			case 1: // save image
				i.Image.Save(i.imageFile, i.shrink)
				i.WaitReply(0, 4)
			case 2: // include file
				i.WaitReply(0, 4)
				f, err := os.Open(i.Image.DecodeString(i.Pop()))
				if err != nil {
					return err
				}
				i.PushInput(f)
			case -1: // open file
				fd := i.openfile(i.Image.DecodeString(i.data[i.sp]), i.Tos)
				i.Drop2()
				i.WaitReply(fd, 4)
			case -2: // read byte
				f := i.files[i.Pop()]
				if f != nil {
					f.Read(b[:])
				}
				i.WaitReply(Cell(b[0]), 4)
			case -3: // write byte
				var l int
				b[0] = byte(i.data[i.sp])
				f := i.files[i.Tos]
				i.Drop2()
				if f != nil {
					l, _ = f.Write(b[:])
				}
				i.WaitReply(Cell(l), 4)
			case -4: // close fd
				var ret Cell = 1
				id := i.Pop()
				if f := i.files[id]; f != nil {
					if err := f.Close(); err == nil {
						i.files[id] = nil
						i.fid = id
						ret = 0
					}
				}
				i.WaitReply(ret, 4)
			case -5: // ftell
				var p int64
				if f := i.files[i.Pop()]; f != nil {
					p, _ = f.Seek(0, 1)
				}
				i.WaitReply(Cell(p), 4)
			case -6: // seek
				var p int64
				o, f := i.data[i.sp], i.files[i.Tos]
				i.Drop2()
				if f != nil {
					p, _ = f.Seek(int64(o), 0)
				}
				i.WaitReply(Cell(p), 4)
			case -7: // file size
				var sz Cell
				if f := i.files[i.Pop()]; f != nil {
					if fi, err := f.Stat(); err == nil {
						sz = Cell(fi.Size())
					}
				}
				i.WaitReply(sz, 4)
			case -8: // delete
				err := os.Remove(i.Image.DecodeString(i.Pop()))
				if err != nil {
					i.WaitReply(0, 4)
				} else {
					i.WaitReply(-1, 4)
				}
			default:
				i.WaitReply(0, 4)
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
				i.Ports[5] = Cell(i.Depth())
			case -6:
				// address depth
				i.Ports[5] = Cell(i.rsp)
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
				src, dst := i.Tos, i.data[i.sp]
				i.Drop2()
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
				i.output.MoveCursor(int(i.Tos), int(i.data[i.sp]))
				i.Drop2()
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
