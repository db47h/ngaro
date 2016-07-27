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
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/pkg/errors"
)

type flusher interface {
	Flush() error
}

type runeWriter interface {
	WriteRune(r rune) (size int, err error)
}

type runeWriterWrapper struct {
	io.Writer
}

func (w *runeWriterWrapper) WriteByte(c byte) (err error) {
	_, err = w.Writer.Write([]byte{c})
	return
}

func (w *runeWriterWrapper) WriteRune(r rune) (size int, err error) {
	b := [utf8.UTFMax]byte{}
	if r < utf8.RuneSelf {
		err = w.WriteByte(byte(r))
		if err != nil {
			return 0, err
		}
		return 1, nil
	}
	l := utf8.EncodeRune(b[:], r)
	return w.Writer.Write(b[0:l])
}

func (w *runeWriterWrapper) Flush() error {
	if f, ok := w.Writer.(flusher); ok {
		return f.Flush()
	}
	return nil
}

// newWriter returns either w if it implements runeWriter or wraps it up into
// a runeWriterWrapper
func newWriter(w io.Writer) runeWriter {
	switch ww := w.(type) {
	case nil:
		return nil
	case runeWriter:
		return ww
	default:
		return &runeWriterWrapper{w}
	}
}

// runeReader wraps a basic reader into a io.RuneReader and io.Closer
type runeReaderWrapper struct {
	io.Reader
}

func (r *runeReaderWrapper) ReadRune() (ret rune, size int, err error) {
	var (
		b = [utf8.UTFMax]byte{}
		i = 0
	)
	for i < utf8.UTFMax && err == nil && !utf8.FullRune(b[:i]) {
		var n int
		n, err = r.Reader.Read(b[i : i+1])
		i += n
	}
	if i == 0 {
		return 0, 0, err
	}
	ret, size = rune(b[0]), 1
	if ret >= utf8.RuneSelf {
		ret, size = utf8.DecodeRune(b[:i])
	}
	return ret, size, err
}

func (r *runeReaderWrapper) Close() error {
	if c, ok := r.Reader.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func newRuneReader(r io.Reader) io.RuneReader {
	switch rr := r.(type) {
	case nil:
		return nil
	case io.RuneReader:
		return rr
	default:
		return &runeReaderWrapper{r}
	}
}

type multiRuneReader struct {
	readers []io.RuneReader
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
		// discard the reader and optionally close it
		if cl, ok := mr.readers[0].(io.Closer); ok {
			cl.Close()
		}
		mr.readers = mr.readers[1:]
	}
	return 0, 0, io.EOF
}

func (mr *multiRuneReader) pushReader(r io.Reader) {
	mr.readers = append([]io.RuneReader{newRuneReader(r)}, mr.readers...)
	for _, rr := range mr.readers {
		fmt.Printf("%#v\n", rr)
	}
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
		r, size, err := i.input.ReadRune()
		if size > 0 {
			i.out(Cell(r), 1)
		} else {
			i.out(utf8.RuneError, 1)
			if err != nil {
				return errors.Wrap(err, "ioWait input")
			}
		}
	}

	// output
	if i.ports[2] == 1 {
		r := rune(i.Pop())
		if i.output != nil {
			_, err := i.output.WriteRune(r)
			if err != nil {
				// p.ip = len(p.mem)
				return errors.Wrap(err, "ioWait output")
			}
		}
		i.out(0, 2)
	}

	// File io
	if i.ports[4] != 0 {
		i.ports[0] = 1
		switch i.ports[4] {
		case 1: // save image
			i.Image.Save(i.imageFile, i.shrink)
			i.ports[4] = 0
		case 2: // include file
			i.ports[4] = 0
			f, err := os.Open(i.Image.DecodeString(int(i.Pop())))
			if err != nil {
				return errors.Wrap(err, "Include failed")
			}
			i.PushInput(f)
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
