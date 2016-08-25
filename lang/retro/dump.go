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

package retro

import (
	"io"
	"strconv"

	"github.com/db47h/ngaro/vm"
)

func dumpSlice(w io.Writer, prefix byte, a []vm.Cell) error {
	var err error
	l := len(a) - 1
	b := make([]byte, 0, 14)
	b = append(b, prefix)
	if l >= 0 {
		for i := 0; i < l; i++ {
			b = strconv.AppendInt(b, int64(int(a[i])), 10)
			b = append(b, ' ')
			_, err = w.Write(b)
			if err != nil {
				return err
			}
			b = b[:0]
		}
		b = strconv.AppendInt(b, int64(int(a[l])), 10)
	}
	_, err = w.Write(b)
	return err
}

// DumpVM dumps the virtual machine stacks and memory image to the specified io.Writer.
func DumpVM(i *vm.Instance, size int, w io.Writer) error {
	err := dumpSlice(w, '\x1C', i.Data())
	if err != nil {
		return err
	}
	err = dumpSlice(w, '\x1D', i.Address())
	if err != nil {
		return err
	}
	return dumpSlice(w, '\x1D', i.Mem[:size])
}
