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

package retro_test

import (
	"bytes"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/db47h/ngaro/asm"
	"github.com/db47h/ngaro/lang/retro"
	"github.com/db47h/ngaro/vm"
	"github.com/pkg/errors"
)

func Test_stringCodec(t *testing.T) {
	s := []byte("Go 1.7 rocks")
	b := make([]vm.Cell, 20)
	var i int
	for i = range b {
		b[i] = -1
	}
	e := retro.StringCodec

	e.Encode(b, 1, s)
	if b[0] != -1 {
		t.Fatal("encode at wrong place")
	}
	for i = range s {
		if vm.Cell(s[i]) != b[i+1] {
			t.Fatalf("Encoding error at pos %d, expected %c, got %c", i+1, s[i], b[i+1])
		}
	}
	if b[len(s)+1] != 0 {
		t.Fatalf("Buffer overrun: expected 0, got %c", b[len(s)+1])
	}

	d := e.Decode(b, 1)
	if string(d) != string(s) {
		t.Fatalf("Decode error. Expected \"%s\", got \"%s\"", s, d)
	}

	if x := e.Decode(b, 100); x != nil {
		t.Fatalf("Decode error. Expected nil, got \"%s\"", x)
	}

	e.Encode(b, 19, []byte("XYZ"))
	if b[19] != 'X' {
		t.Fatalf("Edge encode error. Expected X, got '%c'", b[19])
	}
}

func checkFileSize(fn string, sz int64) error {
	info, err := os.Stat(fn)
	if err != nil {
		return errors.Wrapf(err, "Stat(\"%s\") failed", fn)
	}
	if info.Size() != sz {
		return errors.Errorf("filesize mismatch: Expected %d, got %d", sz, info.Size())
	}
	return nil
}

func saveMemAndCheck(fn string, mem []vm.Cell, shrink bool, cells int) error {
	f := retro.ShrinkSave(shrink, 32)
	err := f(fn, mem)
	if err != nil {
		return errors.Wrap(err, "save failed")
	}
	err = checkFileSize(fn, int64(cells)*4)
	os.Remove(fn)
	if err != nil {
		return errors.Wrap(err, "check failed")
	}
	return nil
}

func TestShrinkSave(t *testing.T) {
	fn := path.Join(os.TempDir(), "testShrink")
	mem := make([]vm.Cell, 20)
	if err := saveMemAndCheck(fn, mem, false, 20); err != nil {
		t.Fatal(err)
	}
	mem[3] = 12
	if err := saveMemAndCheck(fn, mem, true, 12); err != nil {
		t.Fatal(err)
	}
	mem[3] = 144
	if err := saveMemAndCheck(fn, mem, true, 20); err != nil {
		t.Fatal(err)
	}
}

func TestDumpVM(t *testing.T) {
	mem, err := asm.Assemble("testDumpVM", strings.NewReader("nop lit 42"))
	i, err := vm.New(mem, "")
	if err != nil {
		t.Fatal(err)
	}
	i.Push(17)
	var b bytes.Buffer
	err = retro.DumpVM(i, len(i.Mem), &b)
	if err != nil {
		t.Fatal(err)
	}
	exp := "\x1C17\x1D\x1D0 1 42"
	if s := b.String(); s != exp {
		t.Fatalf("Expected:\n%s\ngot: %s", strconv.Quote(exp), strconv.Quote(s))
	}
}
