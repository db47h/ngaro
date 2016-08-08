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
	"io"
	"strconv"
	"unicode/utf8"
)

// readWriter wraps the WriteRune method. Works the same ad bufio.Writer.WriteRune.
type runeWriter interface {
	io.Writer
	WriteRune(r rune) (size int, err error)
}

// runeWriterWrapper wraps a plain io.Reader into a runeWriter.
type runeWriterWrapper struct {
	io.Writer
}

func (w *runeWriterWrapper) WriteRune(r rune) (size int, err error) {
	b := [utf8.UTFMax]byte{}
	if r < utf8.RuneSelf {
		return w.Write([]byte{byte(r)})
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

// Close forwards to the wrapped reader if it implements it.
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

type vt100Terminal struct {
	runeWriter
	flush func() error
	size  func() (int, int)
}

func (t *vt100Terminal) Flush() error {
	if t.flush == nil {
		return nil
	}
	return t.flush()
}
func (t *vt100Terminal) Size() (width int, height int) {
	if t.size == nil {
		return 0, 0
	}
	return t.size()
}
func (t *vt100Terminal) Clear() {
	t.runeWriter.Write([]byte{'\033', '[', '2', 'J', '\033', '[', '1', ';', '1', 'H'})
}
func (t *vt100Terminal) MoveCursor(row, col int) {
	var b bytes.Buffer
	b.Write([]byte{'\033', '['})
	b.Write([]byte(strconv.Itoa(row)))
	b.Write([]byte{';'})
	b.Write([]byte(strconv.Itoa(col)))
	_, err := b.Write([]byte{'H'})
	if err == nil {
		io.Copy(t.runeWriter, &b)
	}
}
func (t *vt100Terminal) FgColor(fg int) {
	t.runeWriter.Write([]byte{'\033', '[', '3', '0' + byte(fg), 'm'})
}
func (t *vt100Terminal) BgColor(bg int) {
	t.runeWriter.Write([]byte{'\033', '[', '4', '0' + byte(bg), 'm'})
}
func (t *vt100Terminal) Port8Enabled() bool { return true }

// NewVT100Terminal returns a new Terminal implementation that uses VT100 escape
// sequences to implement the Clear, CusrosrPos, FgColor and BgColor methods.
//
// The caller only needs to provide the functions implementing Flush and Size.
// Either of these functions may be nil, in which case they will be implemented
// as no-ops.
func NewVT100Terminal(w io.Writer, flush func() error, size func() (width int, height int)) Terminal {
	return &vt100Terminal{newWriter(w), flush, size}
}
