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
// programs via custom I/O handlers (i.e. scripting Go programs from Retro). The
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
// TODO:
//	- complete file i/o
//	- add a reset func: clear stacks/reset ip to 0, accept Options (input / output may need to be reset as well)
//	- add a disassembly function.
//	- go routines
//	- BUG: I/O trashes ports in interactive mode. For example, the following returns 0 instead of the image size:
//		-1 5 out 0 0 out wait 5 in putn
//
package vm
