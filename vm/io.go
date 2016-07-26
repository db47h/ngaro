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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

type multiRuneReader struct {
	readers []RuneReader
}

func (mr *multiRuneReader) ReadRune() (r rune, size int, err error) {
	for len(mr.readers) > 0 {
		r, size, err = mr.readers[0].ReadRune()
		if size > 0 || err != io.EOF {
			if err == io.EOF {
				err = nil
			}
			return
		}
		// dump that reader
		mr.readers = mr.readers[1:]
	}
	return 0, 0, io.EOF
}

func (mr *multiRuneReader) pushReader(r RuneReader) {
	mr.readers = append([]RuneReader{r}, mr.readers...)
}

// PushInput sets r as the current input RuneReader for the VM. When this reader
// reaches EOF, the previously pushed reader will be used.
func (i *Instance) PushInput(r RuneReader) {
	switch in := i.input.(type) {
	case nil:
		fmt.Fprintf(os.Stderr, "Single reader\n")
		i.input = r
	case *multiRuneReader:
		fmt.Fprintf(os.Stderr, "Pushing reader to existing mr\n")
		in.pushReader(r)
	default:
		fmt.Fprintf(os.Stderr, "Pushing reader to new mr\n")
		mr := &multiRuneReader{
			readers: []RuneReader{in},
		}
		mr.pushReader(r)
		i.input = mr
	}
}

type breakError struct{}

func (breakError) Error() string {
	return "user breakpoint"
}

func (i *Instance) out(v Cell, port int) {
	i.ports[port] = v
	i.ports[0] = 1
}

func (i *Instance) ioWait() error {
	if i.ports[0] == 1 {
		return nil
	}

	// input
	if i.ports[1] == 1 {
		r, _, err := i.input.ReadRune()
		if err != nil {
			i.out(-1, 1)
			// p.ip = len(p.mem)
			return errors.Wrap(err, "ioWait input")
		}
		i.out(Cell(r), 1)
	}

	// output
	if i.ports[2] == 1 {
		r := rune(i.Pop())
		_, err := i.output.WriteRune(r)
		if err != nil {
			// p.ip = len(p.mem)
			return errors.Wrap(err, "ioWait output")
		}
		i.out(0, 2)
	}

	// File io
	if i.ports[4] != 0 {
		i.ports[0] = 1
		switch i.ports[4] {
		case 1: // save image
			panic("TODO: Save image")
		case 2: // include file
			f, err := os.Open(i.Image.DecodeString(int(i.Pop())))
			if err != nil {
				return errors.Wrap(err, "Include failed")
			}
			defer f.Close()
			b, err := ioutil.ReadAll(f)
			if err != nil {
				return errors.Wrap(err, "Include failed")
			}
			i.PushInput(bytes.NewBuffer(b))
			i.ports[4] = 0
		default:
			i.ports[4] = 0
		}
	}

	if i.ports[5] != 0 {
		switch i.ports[5] {
		case -1: // mem size
			i.ports[5] = Cell(len(i.Image))
		case -5:
			i.ports[5] = Cell(i.sp + 1)
		case -6:
			i.ports[5] = Cell(i.rsp + 1)
		case -8:
			i.ports[5] = Cell(time.Now().Unix())
		case -9:
			i.ports[5] = 0
			i.ip = len(i.Image) - 1 // will be incremented when returning
		case -13:
			i.ports[5] = Cell(unsafe.Sizeof(i.ports[0]) * 8)
		case -16:
			i.ports[5] = Cell(len(i.data))
		case -17:
			i.ports[5] = Cell(len(i.address))
		default:
			i.ports[5] = 0
		}
		i.ports[0] = 1
	}
	return nil
}
