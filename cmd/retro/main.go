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
	var fileName = flag.String("image", "retroImage", "Use `filename` as the image to load")
	var withFile = flag.String("with", "", "Add `filename` to the input stack")
	var shrink = flag.Bool("shrink", true, "When saving, don't save unused cells")
	flag.Parse()

	// default options
	var optlist = []vm.Option{
		vm.OptShrinkImage(*shrink),
		// buffered io is faster
		vm.OptOutput(bufio.NewWriter(os.Stdout)),
		vm.OptInput(bufio.NewReader(os.Stdin)),
	}

	// append withFile to the input stack
	if len(*withFile) > 0 {
		f, err := os.Open(*withFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		optlist = append(optlist, vm.OptInput(f))
	}

	img, err := vm.Load(*fileName, 1000000)
	if err == nil {
		proc := vm.New(img, *fileName, optlist...)
		err = proc.Run(len(proc.Image))
	}
	if err != nil {
		switch errors.Cause(err) {
		case io.EOF: // stdin or stdout closed
		default:
			fmt.Fprintf(os.Stderr, "%+v\n", err)
			os.Exit(1)
		}
	}
}
