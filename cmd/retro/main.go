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

	"github.com/db47h/ngaro/vm"
)

var fileName = flag.String("image", "retroImage", "Use `filename` as the image to load")
var withFile = flag.String("with", "", "Add `filename` to the input stack")
var shrink = flag.Bool("shrink", true, "When saving, don't save unused cells")
var size = flag.Int("size", 50000, "image size in cells")
var rawIO = flag.Bool("raw", true, "enable raw terminal IO")
var debug = flag.Bool("debug", false, "enable debug diagnostics")

func main() {
	// check exit condition
	var err error
	var proc *vm.Instance
	defer func() {
		if err == nil {
			return
		}
		if !*debug {
			fmt.Fprintf(os.Stderr, "\n%v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "\n%+v\n", err)
		if proc != nil {
			if proc.PC < len(proc.Image) {
				fmt.Fprintf(os.Stderr, "PC: %v (%v), Stack: %v, Ret: %v\n", proc.PC, proc.Image[proc.PC], proc.Data(), proc.Address())
			} else {
				fmt.Fprintf(os.Stderr, "PC: %v, Stack: %v\nRet:  %v\n", proc.PC, proc.Data(), proc.Address())
			}
		}
		os.Exit(1)
	}()

	flag.Parse()

	// try to switch the output terminal to raw mode.
	var rawtty bool
	if *rawIO {
		fn, e := setRawIO()
		if e == nil {
			rawtty = true
			defer fn()
		}
	}

	output := bufio.NewWriter(os.Stdout)
	outTerm := vm.NewVT100Terminal(output, output.Flush, consoleSize(os.Stdout))

	// default options
	var opts = []vm.Option{
		vm.Shrink(*shrink),
		vm.Output(outTerm),
	}

	if rawtty {
		// with the terminal in raw mode, we need to manually handle CTRL-D and
		// backspace, so we'll intercept WAITs on ports 1 and 2.
		// we could also do it with wrappers around Stdin/Stdout
		opts = append(opts,
			vm.Input(os.Stdin),
			vm.BindWaitHandler(1, func(v, port vm.Cell) error {
				if v != 1 {
					return proc.Wait(v, port)
				}
				// if v == 1, this will always read something
				e := proc.Wait(v, port)
				if e == nil && proc.Ports[1] == 4 {
					// CTRL-D
					return io.EOF
				}
				return e

			}),
			vm.BindWaitHandler(2, func(v, port vm.Cell) error {
				var e error
				if v != 1 {
					return proc.Wait(v, port)
				}
				t := proc.Tos() // save TOS
				e = proc.Wait(v, port)
				if e == nil && t == 8 {
					// th vm has written a backspace, erase char under cursor
					_, e = outTerm.Write([]byte{32, 8})
				}
				return e
			}))
	} else {
		// If not raw tty, buffer stdin, but do not check further if the i/o is
		// a terminal or not. The standard VT100 behavior is sufficient here.
		opts = append(opts, vm.Input(bufio.NewReader(os.Stdin)))
	}

	// append withFile to the input stack
	if len(*withFile) > 0 {
		var f *os.File
		f, err = os.Open(*withFile)
		if err != nil {
			return
		}
		opts = append(opts, vm.Input(bufio.NewReader(f)))
	}

	img, err := vm.Load(*fileName, *size)
	if err != nil {
		return
	}
	proc, err = vm.New(img, *fileName, opts...)
	if err != nil {
		return
	}
	if err = proc.Run(); err == io.EOF {
		err = nil
	}
}
