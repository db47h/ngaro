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
	"github.com/pkg/errors"
)

func main() {
	// check exit condition
	var err error
	defer func() {
		if err != nil {
			err = errors.Cause(err)
			fmt.Fprintf(os.Stderr, "%+v\n", err)
			os.Exit(1)
		}
	}()

	var fileName = flag.String("image", "retroImage", "Use `filename` as the image to load")
	var withFile = flag.String("with", "", "Add `filename` to the input stack")
	var shrink = flag.Bool("shrink", true, "When saving, don't save unused cells")
	var size = flag.Int("size", 50000, "image size in cells")
	flag.Parse()

	var interactive bool
	fn, err := setRawIO()
	if err == nil {
		interactive = true
		defer fn()
	}

	// default options
	var opts = []vm.Option{
		vm.Shrink(*shrink),
		// buffered io is faster
		vm.Output(bufio.NewWriter(os.Stdout)),
	}

	// buffer input if not interactive
	if interactive {
		opts = append(opts, vm.Input(os.Stdin))
	} else {
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
	proc, err := vm.New(img, *fileName, opts...)
	if err != nil {
		return
	}
	err = proc.Run(len(proc.Image))
	// filter out EOF
	if e := errors.Cause(err); e == io.EOF {
		err = nil
	}
}
