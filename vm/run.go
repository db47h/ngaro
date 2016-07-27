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

// Push pushes the argument on top of the data stack.
func (i *Instance) Push(v Cell) {
	i.sp++
	i.data[i.sp] = v
}

// Pop pops the value on top of the data stack and returns it.
func (i *Instance) Pop() Cell {
	sp := i.sp
	i.sp--
	return i.data[sp]
}

// Rpush pushes the argument on top of the address stack.
func (i *Instance) Rpush(v Cell) {
	i.rsp++
	i.address[i.rsp] = v
}

// Rpop pops the value on top of the address stack and returns it.
func (i *Instance) Rpop() Cell {
	rsp := i.rsp
	i.rsp--
	return i.address[rsp]
}

// Run starts execution of the VM until the program pointer reaches `toPC`.
// If an error occurs, the PC will will point to the instruction that triggered
// the error.
//
// If the VM was exited cleanly from a user program with the `bye` word, the PC
// will be equal to len(i.Image) and err will be nil.
func (i *Instance) Run(toPC int) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.Errorf("%v", e)
		}
	}()
	i.insCount = 0
	for i.PC < len(i.Image) {
		// fmt.Printf("% 8d\t%s", p.ip, opcode)
		// switch opcode {
		// case OpLit, OpLoop, OpJump, OpGtJump, OpLtJump, OpEqJump, OpNeJump:
		// 	fmt.Printf(" %d", int(p.Image[p.ip+1]))
		// }
		// fmt.Printf("\t%v\n", p.data[0:p.sp+1])
		op := opcode(i.Image[i.PC])
		switch op {
		case OpNop:
			i.PC++
		case OpLit:
			i.Push(i.Image[i.PC+1])
			i.PC += 2
		case OpDup:
			i.Push(i.data[i.sp])
			i.PC++
		case OpDrop:
			i.sp--
			i.PC++
		case OpSwap:
			i.data[i.sp], i.data[i.sp-1] = i.data[i.sp-1], i.data[i.sp]
			i.PC++
		case OpPush:
			i.Rpush(i.Pop())
			i.PC++
		case OpPop:
			i.Push(i.Rpop())
			i.PC++
		case OpLoop:
			v := i.data[i.sp] - 1
			if v > 0 {
				i.data[i.sp] = v
				i.PC = int(i.Image[i.PC+1])
			} else {
				i.sp--
				i.PC += 2
			}
		case OpJump:
			i.PC = int(i.Image[i.PC+1])
		case OpReturn:
			i.PC = int(i.Rpop() + 1)
		case OpGtJump:
			if i.data[i.sp-1] > i.data[i.sp] {
				i.PC = int(i.Image[i.PC+1])
			} else {
				i.PC += 2
			}
			i.sp -= 2
		case OpLtJump:
			if i.data[i.sp-1] < i.data[i.sp] {
				i.PC = int(i.Image[i.PC+1])
			} else {
				i.PC += 2
			}
			i.sp -= 2
		case OpNeJump:
			if i.data[i.sp-1] != i.data[i.sp] {
				i.PC = int(i.Image[i.PC+1])
			} else {
				i.PC += 2
			}
			i.sp -= 2
		case OpEqJump:
			if i.data[i.sp-1] == i.data[i.sp] {
				i.PC = int(i.Image[i.PC+1])
			} else {
				i.PC += 2
			}
			i.sp -= 2
		case OpFetch:
			i.data[i.sp] = i.Image[i.data[i.sp]]
			i.PC++
		case OpStore:
			i.Image[i.data[i.sp]] = i.data[i.sp-1]
			i.sp -= 2
			i.PC++
		case OpAdd:
			rhs := i.Pop()
			i.data[i.sp] += rhs
			i.PC++
		case OpSub:
			rhs := i.Pop()
			i.data[i.sp] -= rhs
			i.PC++
		case OpMul:
			rhs := i.Pop()
			i.data[i.sp] *= rhs
			i.PC++
		case OpDimod:
			lhs, rhs := i.data[i.sp-1], i.data[i.sp]
			i.data[i.sp-1] = lhs % rhs
			i.data[i.sp] = lhs / rhs
			i.PC++
		case OpAnd:
			rhs := i.Pop()
			i.data[i.sp] &= rhs
			i.PC++
		case OpOr:
			rhs := i.Pop()
			i.data[i.sp] |= rhs
			i.PC++
		case OpXor:
			rhs := i.Pop()
			i.data[i.sp] ^= rhs
			i.PC++
		case OpShl:
			rhs := i.Pop()
			i.data[i.sp] <<= uint8(rhs)
			i.PC++
		case OpShr:
			rhs := i.Pop()
			i.data[i.sp] >>= uint8(rhs)
			i.PC++
		case OpZeroExit:
			if i.data[i.sp] == 0 {
				i.PC = int(i.Rpop() + 1)
				i.sp--
			} else {
				i.PC++
			}
		case OpInc:
			i.data[i.sp]++
			i.PC++
		case OpDec:
			i.data[i.sp]--
			i.PC++
		case OpIn:
			port := i.data[i.sp]
			i.data[i.sp], i.ports[port] = i.ports[port], 0
			i.PC++
		case OpOut:
			port := i.data[i.sp]
			i.ports[port] = i.data[i.sp-1]
			i.sp -= 2
			if port == 3 {
				if o, ok := i.output.(flusher); ok {
					o.Flush()
				}
			}
			i.PC++
		case OpWait:
			err = i.ioWait()
			switch err.(type) {
			case nil:
				i.PC++
			default:
				return err
			}
		default:
			i.rsp++
			i.address[i.rsp], i.PC = Cell(i.PC), int(op)
		}
		i.insCount++
	}
	return nil
}
