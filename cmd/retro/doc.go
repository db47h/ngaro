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

// The retro command line tool is a a showcase for the package github.com/db47h/ngaro/vm
// and can be used as a high performance replacement for the Retro reference implementations.
//
// See aslo http://forthworks.com/retro/ for more information on Retro.
//
// Usage:
//
//	-debug
//		  enable debug diagnostics
//	-dump
//		  dump stacks and memory image upon exit, for ngarotest.py
//	-ibits value
//		  cell size in bits of loaded memory image (default GOARCH bits)
//	-image filename
//		  Load memory image from file filename (default "retroImage")
//	-noraw
//		  disable raw terminal IO
//	-noshrink
//		  When saving, don't shrink memory image file
//	-o filename
//		  filename to use when saving memory image
//	-obits value
//		  cell size in bits of saved memory image (default GOARCH bits)
//	-size int
//		  runtime memory image size in cells (default 100000)
//	-with filename
//		  Add filename to the input list (can be specified multiple times)
//
// -debug: will print a full stacktrace should the VM crash.
//
// -dump: this boolean flag is meant to be used in conjonction with the Retro
// test suite. It will dunp the stacks and memory image to stdout.
//
// -noraw: upon startup, retro switches the terminal to raw mode unless stdin
// has been redirected. This flag disables this behavior.
//
// -image: memory image file to load on startup. The default is a file named
// "retroImage" in the current directory.
//
// -size: total memory image size (in cells) to use at runtime. It may be
// automatically extended to fit the loaded memory image file. Make sure that
// this value is sufficiently big to have some free cells as temporary storage.
//
// -with: After loading the memory image, retro will feed the specified file to
// the VM as input. If specified multiple times, files will be fed to the VM in
// order of appearance on the command line.
//
// -o: filename to use when saving the memory image. In Retro, the "save" wword
// saves the memory image to disk. This can also be done in Ngaro assembler with
// the following code:
//
//	.org 32 ( or whatever address above 32 )
//	:save
//		1 4 out
//		0 0 out
//		wait ;
//	save
//
// -noshrink: in the Retro language, the number of cells allocated inside the
// memory image is stored in cell #3. By default, this value is used to save
// only that many cells. Use this flag if you are running a memory image
// incompatible with this scheme, or if you want to make a full snapshot of the
// VM memory, including temp data.
//
// -ibits, -obits: control respectively the cell size in bits of the input and
// output memory images. These flags are primarily meant to convert memory
// images between different cell sizes. For more details on 32/64 bits handling
// and examples, please see https://github.com/db47h/ngaro/blob/master/README.md
package main
