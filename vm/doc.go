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

// Package vm implements the Ngaro VM.
//
// Please visit http://forthworks.com/retro/ to get you started about the Retro
// language and the Ngaro Virtual Machine.
//
// This implementation allows communication between Retro programs and Go
// programs via custom I/O handlers (i.e. scripting Go programs in Retro). The
// package examples demonstrate various use cases. For more details on I/O
// handling in the Ngaro VM, please refer to
// http://retroforth.org/docs/The_Ngaro_Virtual_Machine.html.
//
// For all intents and purposes, the VM behaves according to the specification.
// With one exception: if you venture into hacking the VM code itself, be aware
// that for performance reasons, the PC (aka. Instruction Pointer) is not
// incremented in a single place, rather each opcode deals with the PC as
// needed. This should be of no concern to any other users, even with custom I/O
// handlers. Should you find that the VM does not behave according to the spec,
// please file a bug report.
//
// There's a caveat common to all Ngaro implementations: use of IN, OUT and WAIT
// from the listener (the Retro interactive prompt) will not work as expected.
// This is because the listener uses the same mechanism to read user input and
// write to the terminal and will clear port 0 before you get a chance to
// read/clear response values. This is of particular importance for users of
// custom IO handlers. To work around this issue, a synchronous OUT-WAIT-IN IO
// sequence must be compiled in a word, so that it will run atomically, without
// interference from the listener. For example, to read VM capabilities, you can
// do this:
//
//	( io sends value n to port p, does a wait and puts response back on the stack )
//	: io ( np-n ) dup push out 0 0 out wait pop in ;
//
//	-1 5 io putn
//
// should give you the size of the image.
//
// Since characters are stored one per 32 bits Cell in Retro, the VM supports
// unicode I/O. If the io.Reader and io.Writer provided as input implement the
// ReadRune and WriteRune methods respectively, these methods will be used for
// I/O.
//
// Regarding I/O, reading console width and height will only work if the
// io.Writer set as output with vm.Output implements the Fd method. So this will
// only work if the output is os.Stdout or a pty (and NOT wrapped in a
// bufio.Writer).
//
// TODO:
//	- complete file i/o
//	- add a reset func: clear stacks/reset ip to 0, accept Options (input / output may need to be reset as well)
//	- go routines in cmd/retro
//	- implement stats and debug image dump
package vm
