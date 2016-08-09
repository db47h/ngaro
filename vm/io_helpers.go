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
)

type multiReader struct {
	readers []io.Reader
}

func (mr *multiReader) Read(p []byte) (n int, err error) {
	for len(mr.readers) > 0 {
		n, err = mr.readers[0].Read(p)
		if n > 0 || err != io.EOF {
			if err == io.EOF {
				// Don't return EOF yet. There may be more bytes
				// in the remaining readers.
				err = nil
			}
			return
		}
		if c, ok := mr.readers[0].(io.Closer); ok {
			c.Close()
		}
		mr.readers = mr.readers[1:]
	}
	return 0, io.EOF
}

func (mr *multiReader) pushReader(r io.Reader) {
	mr.readers = append([]io.Reader{r}, mr.readers...)
}

type vt100Terminal struct {
	io.Writer
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
	t.Write([]byte{'\033', '[', '2', 'J', '\033', '[', '1', ';', '1', 'H'})
}
func (t *vt100Terminal) MoveCursor(row, col int) {
	var b bytes.Buffer
	b.Write([]byte{'\033', '['})
	b.Write([]byte(strconv.Itoa(row)))
	b.Write([]byte{';'})
	b.Write([]byte(strconv.Itoa(col)))
	_, err := b.Write([]byte{'H'})
	if err == nil {
		io.Copy(t, &b)
	}
}
func (t *vt100Terminal) FgColor(fg int) {
	t.Write([]byte{'\033', '[', '3', '0' + byte(fg), 'm'})
}
func (t *vt100Terminal) BgColor(bg int) {
	t.Write([]byte{'\033', '[', '4', '0' + byte(bg), 'm'})
}
func (t *vt100Terminal) Port8Enabled() bool { return true }

// NewVT100Terminal returns a new Terminal implementation that uses VT100 escape
// sequences to implement the Clear, CusrosrPos, FgColor and BgColor methods.
//
// The caller only needs to provide the functions implementing Flush and Size.
// Either of these functions may be nil, in which case they will be implemented
// as no-ops.
func NewVT100Terminal(w io.Writer, flush func() error, size func() (width int, height int)) Terminal {
	return &vt100Terminal{w, flush, size}
}
