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

// Package vm provides an embeddable Ngaro Virtual Machine implementation.
//
// Please visit http://forthworks.com/retro/ to get you started about the Retro
// language and the Ngaro Virtual Machine.
//
// The main purpose of this implementation is to allow customization and
// communication between Retro programs and Go programs via custom opcodes and
// I/O handlers (i.e. scripting Go programs in Retro). The package examples
// demonstrate various use cases. For more details on I/O handling in the Ngaro
// VM, please refer to
// http://retroforth.org/docs/The_Ngaro_Virtual_Machine.html.
//
// Another goal is to make the VM core as neutral as possible regarding the higher
// level language running on it. Some Retro specific behaviors have been moved from
// the VM to the retro command line tool, like shrinking memory image dumps that
// relies on reading the memory image size at a specific location in memory.
//
// Custom opcodes are implemented by intercepting implicit calls to negative memory
// addresses. This allows the VM to be fully backwards compatible with existing
// Retro images while still providing enhanced capabilities. The maximum number of
// addressable cells is 2^31 when running in 32 bits mode (that's 8GiB or memory on
// the host). The range [-2^31 - 1, -1] is available for custom opcodes.
//
// This implementation passes all tests from the retro-language test suite and
// its performance when running tests/core.rx is slightly better than with the
// reference implementations:
//
//	1.12s for this implementation, no custom opcodes, compiled with Go 1.7.
//	1.30s for the reference Go implementation, compiled with Go 1.7
//	2.22s for the reference C implementation, compiled with gcc-5.4 -O3 -fomit-frame-pointer
//
// For all intents and purposes, the VM behaves according to the specification.
// This is of particular importance to implementors of custom opcodes: the VM
// always increments the PC after each opcode, thus opcodes altering the PC must
// adjust it accordingly (i.e. set it to its real target minus one).
//
// There's a caveat common to all Ngaro implementations: use of in, out and wait
// from the listener (the Retro interactive prompt) will not work as expected.
// This is because the listener will out/wait/in on the same ports as you do
// before you get a chance to read response values. This is of particular
// importance to users of custom IO handlers while testing: a value sitting in
// a control port can cause havok if not read and cleared in between two waits.
// To work around this issue, a synchronous OUT-WAIT-IN IO sequence must be
// compiled in a word, so that it will run atomically without interference from
// the listener. For example:
//
//	( io sends value n to port p, does a wait and puts response back on the stack.
//	  Note that the wait word does an `out 0 0` before issuing the real wait instruction )
//	: io ( np-n ) dup push out wait pop in ;
//
//	-1 5 io putn
//
// should give you the total memory size.
package vm
