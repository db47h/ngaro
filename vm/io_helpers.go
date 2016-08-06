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
	"syscall"
	"unicode/utf8"
)

// fder wraps the Fd method
type fder interface {
	Fd() uintptr
}

// readWriter wraps the WriteRune method. Works the same ad bufio.Writer.WriteRune.
type runeWriter interface {
	WriteRune(r rune) (size int, err error)
}

// runeWriterWrapper wraps a plain io.Reader into a runeWriter.
type runeWriterWrapper struct {
	io.Writer
}

// UnwrapFd checks if the wrapped io.Writer implements fder and returns its Fd
// method or nil.
func (w *runeWriterWrapper) UnwrapFd() func() uintptr {
	if f, ok := w.Writer.(fder); ok {
		return f.Fd
	}
	return nil
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
}

// ioctl to control i.output
// TODO: Need to test ioctl on windows
func ioctl(w runeWriter, request, argp uintptr) (err error) {
	var fd func() uintptr
	switch o := w.(type) {
	case *runeWriterWrapper:
		fd = o.UnwrapFd()
	case fder:
		fd = o.Fd
	default:
		err = syscall.Errno(syscall.EBADF)
	}
	if fd != nil {
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd(), request, argp)
		if errno != 0 {
			err = errno
		}
	}
	return err
}
