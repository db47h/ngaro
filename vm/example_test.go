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

package vm_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/db47h/ngaro/vm"
)

// Shows how to load an image, setup the VM with multiple readers/init code.
func ExampleInstance_Run() {
	imageFile := "testdata/retroImage"
	img, err := vm.Load(imageFile, 50000)
	if err != nil {
		panic(err)
	}

	// output capture buffer
	output := bytes.NewBuffer(nil)

	// Setup the VM instance with os.Stdin as first reader, and we push another
	// reader with some custom init code that will include and run the retro core tests.
	i, err := vm.New(img, imageFile,
		vm.Input(os.Stdin),
		vm.Input(strings.NewReader("\"testdata/core.rx\" :include\n")),
		vm.Output(output, false))

	// run it
	if err == nil {
		err = i.Run()
	}
	if err != nil {
		// in interactive use, err may be io.EOF if any of the IO channels gets closed
		// in which case this would be a normal exit condition
		panic(err)
	}

	// filter output to get the retro core test results.
	b := bytes.Split(output.Bytes(), []byte{'\n'})
	fmt.Printf("%s\n", b[len(b)-5])
	fmt.Printf("%s\n", b[len(b)-4])

	// Output:
	// 360 tests run: 360 passed, 0 failed.
	// 186 words checked, 0 words unchecked, 37 i/o words ignored.
}

// Shows how to setup a port handler.
func ExampleOutHandler() {
	imageFile := "testdata/retroImage"
	img, err := vm.Load(imageFile, 0)
	if err != nil {
		panic(err)
	}

	// we will use a buffered stdout
	output := bufio.NewWriter(os.Stdout)
	// so according to the spec, we should flush the output as soon as port 3
	// is written to:
	outputHandler := func(v vm.Cell) (vm.Cell, error) {
		output.Flush()
		return 0, nil
	}

	i, err := vm.New(img, imageFile,
		vm.Input(strings.NewReader("6 7 * putn\n")),
		vm.Output(output, false),
		vm.OutHandler(3, outputHandler))
	if err != nil {
		panic(err)
	}

	if err = i.Run(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}
	// Retro 11.7.1
	//
	// ok  6
	// ok  7
	// ok  *
	// ok  putn 42
	// ok
}
