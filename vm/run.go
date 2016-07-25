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

import "github.com/pkg/errors"

func (p *Instance) push(v Cell) {
	p.sp++
	p.data[p.sp] = v
}

func (p *Instance) pop() Cell {
	sp := p.sp
	p.sp--
	return p.data[sp]
}

func (p *Instance) rpush(v Cell) {
	p.rsp++
	p.address[p.rsp] = v
}

func (p *Instance) rpop() Cell {
	rsp := p.rsp
	p.rsp--
	return p.address[rsp]
}

// Run starts execution of the VM
func (p *Instance) Run(toIP int) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.Errorf("%v", e)
		}
	}()

	for p.ip < toIP {
		op := opcode(p.Image[p.ip])
		// fmt.Printf("% 8d\t%s", p.ip, opcode)
		// switch opcode {
		// case OpLit, OpLoop, OpJump, OpGtJump, OpLtJump, OpEqJump, OpNeJump:
		// 	fmt.Printf(" %d", int(p.Image[p.ip+1]))
		// }
		// fmt.Printf("\t%v\n", p.data[0:p.sp+1])

		switch op {
		case OpNop:
			p.ip++
		case OpLit:
			p.push(p.Image[p.ip+1])
			p.ip += 2
		case OpDup:
			p.push(p.data[p.sp])
			p.ip++
		case OpDrop:
			p.sp--
			p.ip++
		case OpSwap:
			p.data[p.sp], p.data[p.sp-1] = p.data[p.sp-1], p.data[p.sp]
			p.ip++
		case OpPush:
			p.rpush(p.pop())
			p.ip++
		case OpPop:
			p.push(p.rpop())
			p.ip++
		case OpLoop:
			v := p.data[p.sp] - 1
			if v > 0 {
				p.data[p.sp] = v
				p.ip = int(p.Image[p.ip+1])
			} else {
				p.sp--
				p.ip += 2
			}
		case OpJump:
			p.ip = int(p.Image[p.ip+1])
		case OpReturn:
			p.ip = int(p.rpop() + 1)
		case OpGtJump:
			if p.data[p.sp-1] > p.data[p.sp] {
				p.ip = int(p.Image[p.ip+1])
			} else {
				p.ip += 2
			}
			p.sp -= 2
		case OpLtJump:
			if p.data[p.sp-1] < p.data[p.sp] {
				p.ip = int(p.Image[p.ip+1])
			} else {
				p.ip += 2
			}
			p.sp -= 2
		case OpNeJump:
			if p.data[p.sp-1] != p.data[p.sp] {
				p.ip = int(p.Image[p.ip+1])
			} else {
				p.ip += 2
			}
			p.sp -= 2
		case OpEqJump:
			if p.data[p.sp-1] == p.data[p.sp] {
				p.ip = int(p.Image[p.ip+1])
			} else {
				p.ip += 2
			}
			p.sp -= 2
		case OpFetch:
			p.data[p.sp] = p.Image[p.data[p.sp]]
			p.ip++
		case OpStore:
			p.Image[p.data[p.sp]] = p.data[p.sp-1]
			p.sp -= 2
			p.ip++
		case OpAdd:
			rhs := p.pop()
			p.data[p.sp] += rhs
			p.ip++
		case OpSub:
			rhs := p.pop()
			p.data[p.sp] -= rhs
			p.ip++
		case OpMul:
			rhs := p.pop()
			p.data[p.sp] *= rhs
			p.ip++
		case OpDimod:
			lhs, rhs := p.data[p.sp-1], p.data[p.sp]
			p.data[p.sp-1] = lhs % rhs
			p.data[p.sp] = lhs / rhs
			p.ip++
		case OpAnd:
			rhs := p.pop()
			p.data[p.sp] &= rhs
			p.ip++
		case OpOr:
			rhs := p.pop()
			p.data[p.sp] |= rhs
			p.ip++
		case OpXor:
			rhs := p.pop()
			p.data[p.sp] ^= rhs
			p.ip++
		case OpShl:
			rhs := p.pop()
			p.data[p.sp] <<= UCell(rhs)
			p.ip++
		case OpShr:
			rhs := p.pop()
			p.data[p.sp] >>= UCell(rhs)
			p.ip++
		case OpZeroExit:
			if p.data[p.sp] == 0 {
				p.ip = int(p.rpop() + 1)
				p.sp--
			} else {
				p.ip++
			}
		case OpInc:
			p.data[p.sp]++
			p.ip++
		case OpDec:
			p.data[p.sp]--
			p.ip++
		case OpIn:
			port := p.data[p.sp]
			p.data[p.sp], p.ports[port] = p.ports[port], 0
			p.ip++
		case OpOut:
			port := p.data[p.sp]
			p.ports[port] = p.data[p.sp-1]
			p.sp -= 2
			if port == 3 {
				p.output.Flush()
			}
			p.ip++
		case OpWait:
			if err = p.ioWait(); err != nil {
				return err
			}
			p.ip++
		default:
			p.rsp++
			p.address[p.rsp], p.ip = Cell(p.ip), int(op)
		}
	}
	return nil
}
