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

package vm

import (
	"encoding/binary"
	"os"
	"unsafe"

	"github.com/pkg/errors"
)

// Image encapsulates a VM's memory
type Image []Cell

// Load loads an image from file fileName. The image size will be the largest of
// the file size and minSize parameter.
func Load(fileName string, minSize int) (Image, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "Load")
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "Load")
	}
	sz := st.Size()
	if sz > int64((^uint(0))>>1) { // MaxInt
		return nil, errors.Errorf("Load %v: file too large", fileName)
	}
	var t Cell
	sz /= int64(unsafe.Sizeof(t))
	fileCells := sz
	if int64(minSize) > sz {
		sz = int64(minSize)
	}
	i := make(Image, sz)
	err = binary.Read(f, binary.LittleEndian, i[:fileCells])
	if err != nil {
		return nil, errors.Wrap(err, "Load")
	}
	return i, nil
}

// DecodeString returns the 0 terminated string starting at position pos in the image.
func (i Image) DecodeString(pos int) string {
	end := pos
	for ; end < len(i) && i[end] != 0; end++ {
	}
	str := make([]rune, end-pos)
	for idx, c := range i[pos:end] {
		str[idx] = rune(c)
	}
	return string(str)
}
