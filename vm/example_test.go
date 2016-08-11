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
		vm.Output(vm.NewVT100Terminal(output, nil, nil)))

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

// Shows a common use of OUT port handlers.
func ExampleBindOutHandler() {
	imageFile := "testdata/retroImage"
	img, err := vm.Load(imageFile, 0)
	if err != nil {
		panic(err)
	}

	// Our out handler, will just take the written value, and return its square.
	// The result will be stored in the bound port, so we can read it back with
	// IN.
	outputHandler := func(i *vm.Instance, v, port vm.Cell) error {
		i.Ports[port] = v * v
		return nil
	}
	// Create the VM instance with our port handler bound to port 42.
	// We do not wire any output, we'll just read the results from the stack.
	i, err := vm.New(img, imageFile,
		vm.Input(strings.NewReader(": square 42 out 42 in ; 7 square bye\n")),
		vm.BindOutHandler(42, outputHandler))
	if err != nil {
		panic(err)
	}

	if err = i.Run(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}

	fmt.Println(i.Data())

	// Output:
	// [49]
}

// A simple WAIT handler that overrides the default implementation. It's used
// here to implement a (dummy) canvas. We'll need to override port 5 in order to
// report canvas availability and its size and implement the actual drawing on
// port 6. See http://retroforth.org/docs/The_Ngaro_Virtual_Machine.html
func ExampleBindWaitHandler() {
	imageFile := "testdata/retroImage"
	img, err := vm.Load(imageFile, 50000)
	if err != nil {
		panic(err)
	}

	waitHandler := func(i *vm.Instance, v, port vm.Cell) error {
		switch port {
		case 5: // override VM capabilities
			switch v {
			case -2:
				i.WaitReply(-1, port)
			case -3:
				i.WaitReply(1920, port)
			case -4:
				i.WaitReply(1080, port)
			default:
				// not a value that we handle ourselves, hand over the request
				// to the default implementaion
				return i.Wait(v, port)
			}
			return nil
		case 6: // implement the canvas
			switch v {
			case 1:
				/* color */ _ = i.Pop()
			case 2:
				/* y */ _ = i.Pop()
				/* x */ _ = i.Pop()
				// draw a pixel at x, y...
			case 3:
				// more to implement...
			}
			// complete the request
			i.WaitReply(0, port)
		}
		return nil
	}

	// no output set as we don't care.
	// out program first requests the VM size just to check that our override of
	// port 5 properly hands over unknown requests to the default implementation.
	i, err := vm.New(img, imageFile,
		vm.Input(strings.NewReader(
			": cap ( n-n ) 5 out 0 0 out wait 5 in ;\n"+
				"-1 cap -2 cap -3 cap -4 cap bye\n")),
		vm.BindWaitHandler(5, waitHandler),
		vm.BindWaitHandler(6, waitHandler))
	if err != nil {
		panic(err)
	}

	if err = i.Run(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}

	fmt.Println(i.Data())

	// Output:
	// [50000 -1 1920 1080]
}

// A more complex example of WAIT port handlers to communicate with Go. In this
// example, we use a pair of handlers: a request handler that will initiate a
// backround job, and a result handler to query and wait for the result.
func ExampleBindWaitHandler_async() {
	imageFile := "testdata/retroImage"
	img, err := vm.Load(imageFile, 0)
	if err != nil {
		panic(err)
	}

	// we use a channel map to associate a task ID with a Go channel.
	channels := make(map[vm.Cell]chan vm.Cell)

	// our background task.
	fib := func(v vm.Cell, c chan<- vm.Cell) {
		var v0, v1 vm.Cell = 0, 1
		for v > 1 {
			v0, v1 = v1, v0+v1
			v--
		}
		c <- v1
	}

	// The request hqndler will be bound to port 1000. Write 1 to this port with
	// any arguments on the stack, then do a WAIT.
	// It will respond on the same channel with a task ID.
	execHandler := func(i *vm.Instance, v, port vm.Cell) error {
		// find an unused job id
		idx := vm.Cell(len(channels)) + 1
		// make i/o channel and save it
		c := make(chan vm.Cell)
		channels[idx] = c
		// launch job
		go fib(i.Pop(), c)
		// respond with channel ID
		i.WaitReply(idx, port)
		return nil
	}

	// The result handler will be wired to port 1001. Write the job id to this
	// port followed by a wait and get the result with:
	//
	//	1001 IN
	resultHandler := func(i *vm.Instance, v, port vm.Cell) error {
		c := channels[v]
		if c == nil {
			// no such channel. No need to error: if we do not reply, port 0
			// will be 0, so client code should check port 0 as well.
			return nil
		}
		i.WaitReply(<-c, port)
		delete(channels, v)
		return nil
	}

	// No output, we'll just grab the values from the stack on exit. Note that
	// the port communication MUST be compiled in words (here fibGo and fibGet).
	// Issuing IN/OUT/WAIT from the listener would fail because of interference
	// from the I/O code.
	i, err := vm.New(img, imageFile,
		vm.Input(strings.NewReader(
			`: fibGo ( n-ID ) 1 1000 out 0 0 out wait 1000 in ;
			 : fibGet ( ID-n ) 1001 out 0 0 out wait 1001 in ;
			 46 fibGo fibGet bye `)),
		vm.BindWaitHandler(1000, execHandler),
		vm.BindWaitHandler(1001, resultHandler))
	if err != nil {
		panic(err)
	}

	if err = i.Run(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}

	// So, what's Fib(46)?
	fmt.Println(i.Data())

	// Output:
	// [1836311903]
}

// Disassemble is pretty straightforward. Here we Disassemble a hand crafted
// fibonacci function.
func ExampleImage_Disassemble() {
	var fib vm.Image = []vm.Cell{
		vm.OpPush,
		vm.OpLit, 0,
		vm.OpLit, 1,
		vm.OpPop,
		vm.OpJump, 15, // junp to loop
		vm.OpPush, // save count
		vm.OpDup,
		vm.OpPush, // save n-1
		vm.OpAdd,  // n-2 + n-1
		vm.OpPop,
		vm.OpSwap, // stack: n-2 n-1
		vm.OpPop,
		vm.OpLoop, 8, // loop
		vm.OpSwap,
		vm.OpDrop,
		vm.OpReturn,
		vm.OpLit, // Error: unterminated LIT at the end of imah
	}

	for i := 0; i < len(fib); {
		var d string
		i, d = fib.Disassemble(i)
		fmt.Printf("% 4d\t%s\n", i, d)
	}

	// Output:
	//    1	push
	//    3	0
	//    5	1
	//    6	pop
	//    8	jump 15
	//    9	push
	//   10	dup
	//   11	push
	//   12	+
	//   13	pop
	//   14	swap
	//   15	pop
	//   17	loop 8
	//   18	swap
	//   19	drop
	//   20	;
	//   21	???
}
