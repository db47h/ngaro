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
	"io"
	"strconv"

	"github.com/db47h/ngaro/internal/ngi"
	"github.com/db47h/ngaro/vm"
)

func dumpSlice(w *ngi.ErrWriter, a []vm.Cell) error {
	l := len(a) - 1
	if l >= 0 {
		for i := 0; i < l; i++ {
			io.WriteString(w, strconv.Itoa(int(a[i])))
			w.Write([]byte{' '})
		}
		io.WriteString(w, strconv.Itoa(int(a[l])))
	}
	return w.Err
}

// Dump dumps the virtual machine stacks and memory image to the specified io.Writer.
func dumpVM(i *vm.Instance, size int, w io.Writer) error {
	ew := ngi.NewErrWriter(w)
	ew.Write([]byte{'\x1C'})
	dumpSlice(ew, i.Data())
	ew.Write([]byte{'\x1D'})
	dumpSlice(ew, i.Address())
	ew.Write([]byte{'\x1D'})
	return dumpSlice(ew, i.Mem[:size])
}
