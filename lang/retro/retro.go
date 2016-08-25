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

// Package retro provides utility functions and types that enables running a
// Ngaro Virtual Machine specific to the retro language.
package retro

import (
	"github.com/db47h/ngaro/vm"
)

// StringCodec implements the vm.Codec interface for reading/writing strings in
// Retro memory Inages.
//
// Decode returns the string starting at position start in the specified
// slice. Strings stored in the slice must be zero terminated. The trailing '\0'
// is not returned.
//
// Encode writes the given string at position start in specified slice
// and terminates it with a '\0' vm.Cell.
var StringCodec stringCodec

type stringCodec struct{}

func (stringCodec) Decode(mem []vm.Cell, start vm.Cell) []byte {
	if start < 0 || int(start) >= len(mem) {
		return nil
	}
	var str []byte
	for _, c := range mem[start:] {
		if c == 0 {
			break
		}
		str = append(str, byte(c))
	}
	return str
}

func (stringCodec) Encode(mem []vm.Cell, start vm.Cell, s []byte) {
	pos := int(start)
	for _, c := range s {
		if pos >= len(mem) {
			break
		}
		mem[pos] = vm.Cell(c)
		pos++
	}
	if pos < len(mem) {
		mem[pos] = 0
	}
}

// ShrinkSave returns a closure to pass to vm.SaveMemoryImage that will save
// only the used part of a Retro memory image (i.e. mem[0:HERE]) if shrink is
// true. The cellBits parameter specifies the Cell size in bits to use when
// saving.
func ShrinkSave(shrink bool, cellBits int) func(fileName string, mem []vm.Cell) error {
	return func(fileName string, mem []vm.Cell) error {
		l := vm.Cell(len(mem))
		here := l
		if shrink && len(mem) >= 3 {
			here = mem[3]
		}
		if here < 0 || here > l {
			here = l
		}
		return vm.Save(fileName, mem[:here], cellBits)
	}
}
