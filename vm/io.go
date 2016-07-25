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
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

func (p *Instance) out(v Cell, port int) {
	p.ports[port] = v
	p.ports[0] = 1
}

func (p *Instance) ioWait() error {
	if p.ports[0] == 1 {
		return nil
	}

	// input
	if p.ports[1] == 1 {
		r, _, err := p.input.ReadRune()
		if err != nil {
			p.out(-1, 1)
			// p.ip = len(p.mem)
			return errors.Wrap(err, "ioWait input")
		}
		p.out(Cell(r), 1)
	}

	// output
	if p.ports[2] == 1 {
		r := rune(p.pop())
		_, err := p.output.WriteRune(r)
		if err != nil {
			// p.ip = len(p.mem)
			return errors.Wrap(err, "ioWait output")
		}
		p.out(0, 2)
	}

	// File io
	if p.ports[4] != 0 {

	}

	if p.ports[5] != 0 {
		switch p.ports[5] {
		case -1: // mem size
			p.ports[5] = Cell(len(p.Image))
		case -5:
			p.ports[5] = Cell(p.sp + 1)
		case -6:
			p.ports[5] = Cell(p.rsp + 1)
		case -8:
			p.ports[5] = Cell(time.Now().Unix())
		case -9:
			p.ports[5] = 0
			p.ip = len(p.Image)
		case -13:
			p.ports[5] = Cell(unsafe.Sizeof(p.ports[0]) * 8)
		case -16:
			p.ports[5] = Cell(len(p.data))
		case -17:
			p.ports[5] = Cell(len(p.address))
		default:
			p.ports[5] = 0
		}
		p.ports[0] = 1
	}
	return nil
}
