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
	"fmt"
	"io"
	"os"
	"time"

	"github.com/db47h/ngaro/vm"
	"github.com/pkg/errors"
)

func main() {
	fileName := "retroImage"
	img, err := vm.Load(fileName, 1000000)
	if err == nil {
		n := time.Now()
		proc := vm.New(img, fileName)
		_, err = proc.Run(1000000)
		el := time.Now().Sub(n).Seconds()
		c := proc.InstructionCount()
		fmt.Fprintf(os.Stderr, "Executed %d instructions in %.3fs. Perf: %.2f MIPS\n", c, el, float64(c)/1e6/el)
	}
	if err != nil {
		switch errors.Cause(err) {
		case io.EOF: // stdin or stdout closed
		default:
			fmt.Fprintf(os.Stderr, "%+v\n", err)
			os.Exit(1)
		}
	}
}
