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

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"unsafe"

	"github.com/db47h/ngaro/vm"
)

type fileList []string

func (f *fileList) String() string     { return "" }
func (f *fileList) Set(s string) error { *f = append(*f, s); return nil }
func (f *fileList) Get() interface{}   { return *f }

type cellSizeBits int

func (sz *cellSizeBits) String() string { return strconv.Itoa(int(*sz)) }
func (sz *cellSizeBits) Set(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	switch n {
	case 32, 64:
		*sz = cellSizeBits(n)
		return nil
	default:
		return fmt.Errorf("%d bits cells not supported", n)
	}
}
func (sz *cellSizeBits) Get() interface{} { return *sz }

var (
	noShrink    bool
	noRawIO     bool
	debug       bool
	dump        bool
	outFileName string
	srcCellSz   = cellSizeBits(unsafe.Sizeof(vm.Cell(0)) * 8)
	dstCellSz   = srcCellSz
)

func port1Handler(i *vm.Instance, v, port vm.Cell) error {
	if v != 1 {
		return i.Wait(v, port)
	}
	// if v == 1, this will always read something
	e := i.Wait(v, port)
	// in raw tty mode, we need to handle CTRL-D ourselves
	if e == nil && i.Ports[1] == 4 {
		return io.EOF
	}
	return e
}

func port2Handler(w io.Writer) func(i *vm.Instance, v, port vm.Cell) error {
	return func(i *vm.Instance, v, port vm.Cell) error {
		var e error
		if v != 1 {
			return i.Wait(v, port)
		}
		t := i.Tos // save TOS
		e = i.Wait(v, port)
		if e == nil && t == 8 && i.Ports[port] == 0 {
			// the vm has written a backspace, erase char under cursor
			_, e = w.Write([]byte{32, 8})
		}
		return e
	}
}

// Save handler to enable shrinking of retro images
func shrinkSave(fileName string, mem []vm.Cell, _ int) error {
	end := vm.Cell(len(mem))
	if len(mem) < 4 {
		return nil
	}
	if here := mem[3]; !noShrink && here >= 0 && here < end {
		end = here
	}
	err := vm.Save(outFileName, mem[:end], int(dstCellSz))
	if err != nil {
		return err
	}
	return nil
}

func setupIO() (raw bool, tearDown func()) {
	var err error
	if !noRawIO {
		tearDown, err = setRawIO()
		if err != nil {
			return false, nil
		}
	}
	return true, tearDown
}

func newVM(name, saveName string, size, cellSize int, opts ...vm.Option) (*vm.Instance, int, error) {
	mem, fileCells, err := vm.Load(name, size, cellSize)
	if err != nil {
		return nil, fileCells, err
	}
	i, err := vm.New(mem, saveName, opts...)
	return i, fileCells, err
}

func atExit(i *vm.Instance, err error) {
	if err == nil {
		return
	}
	if !debug {
		fmt.Fprintf(os.Stderr, "\n%v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "\n%+v\n", err)
	if i != nil {
		if i.PC < len(i.Mem) {
			fmt.Fprintf(os.Stderr, "PC: %v (%v), Stack: %v, Addr: %v\n", i.PC, i.Mem[i.PC], i.Data(), i.Address())
		} else {
			fmt.Fprintf(os.Stderr, "PC: %v, Stack: %v\nAddr:  %v\n", i.PC, i.Data(), i.Address())
		}
	}
	os.Exit(1)
}

func main() {
	// check exit condition
	var err error
	var i *vm.Instance
	var fileCells int

	stdout := bufio.NewWriter(os.Stdout)
	output := vm.NewVT100Terminal(stdout, stdout.Flush, consoleSize(os.Stdout))

	// flush output, catch and log errors
	defer func() {
		output.Flush()
		if err == nil && dump {
			err = dumpVM(i, fileCells, os.Stdout)
		}
		atExit(i, err)
	}()

	var withFiles fileList

	var fileName = flag.String("image", "retroImage", "Load memory image from file `filename`")
	flag.Var(&srcCellSz, "ibits", "cell size in bits of loaded memory image")
	var size = flag.Int("size", 100000, "image size in cells")
	flag.BoolVar(&dump, "dump", false, "dump stacks and image upon exit, for ngarotest.py")
	flag.Var(&withFiles, "with", "Add `filename` to the input list (can be specified multiple times)")
	flag.BoolVar(&noShrink, "noshrink", false, "When saving, don't shrink image")
	flag.BoolVar(&noRawIO, "noraw", false, "disable raw terminal IO")
	flag.BoolVar(&debug, "debug", false, "enable debug diagnostics")
	flag.StringVar(&outFileName, "o", "", "`filename` to use when saving memory image")
	flag.Var(&dstCellSz, "obits", "cell size in bits of saved image")

	flag.Parse()

	// try to switch the output terminal to raw mode.
	rawtty, ioTearDownFn := setupIO()
	if ioTearDownFn != nil {
		defer ioTearDownFn()
	}

	// default options
	var opts = []vm.Option{
		vm.SaveMemImage(shrinkSave),
		vm.Output(output),
	}

	if rawtty {
		// with the terminal in raw mode, we need to manually handle CTRL-D and
		// backspace, so we'll intercept WAITs on ports 1 and 2.
		// we could also do it with wrappers around Stdin/Stdout
		opts = append(opts,
			vm.Input(os.Stdin),
			vm.BindWaitHandler(1, port1Handler),
			vm.BindWaitHandler(2, port2Handler(output)))
	} else {
		// If not raw tty, buffer stdin, but do not check further if the i/o is
		// a terminal or not. The standard VT100 behavior is sufficient here.
		opts = append(opts, vm.Input(bufio.NewReader(os.Stdin)))
	}

	// append -with files to input stack in reverse order so that they load
	// in order of appearance on the command line.
	for n := len(withFiles) - 1; n >= 0; n-- {
		var f *os.File
		f, err = os.Open(withFiles[n])
		if err != nil {
			return
		}
		opts = append(opts, vm.Input(bufio.NewReader(f)))
	}

	if outFileName == "" {
		outFileName = *fileName
	}
	i, fileCells, err = newVM(*fileName, outFileName, *size, int(srcCellSz), opts...)
	if err != nil {
		return
	}
	if err = i.Run(); err == io.EOF {
		err = nil
	}
}
