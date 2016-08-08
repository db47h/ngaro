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

// Ngaro Virtual Machine Opcodes.
const (
	OpNop Cell = iota
	OpLit
	OpDup
	OpDrop
	OpSwap
	OpPush
	OpPop
	OpLoop
	OpJump
	OpReturn
	OpGtJump
	OpLtJump
	OpNeJump
	OpEqJump
	OpFetch
	OpStore
	OpAdd
	OpSub
	OpMul
	OpDimod
	OpAnd
	OpOr
	OpXor
	OpShl
	OpShr
	OpZeroExit
	OpInc
	OpDec
	OpIn
	OpOut
	OpWait
)

// Tos returns the top stack item.
func (i *Instance) Tos() Cell {
	return i.data[i.sp]
}

// Drop removes the top item from the data stack.
func (i *Instance) Drop(v Cell) {
	i.sp--
}

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

// Run starts execution of the VM.
//
// If an error occurs, the PC will will point to the instruction that triggered
// the error.
//
// If the VM was exited cleanly from a user program with the `bye` word, the PC
// will be equal to len(i.Image) and err will be nil.
//
// If the last input stream gets closed, the VM will exit and return io.EOF.
// This is a normal exit condition in most use cases.
func (i *Instance) Run() (err error) {
	defer func() {
		if e := recover(); e != nil {
			switch e := e.(type) {
			case error:
				err = errors.Wrap(e, "Recovered error")
			default:
				panic(e)
			}
		}
	}()
	i.insCount = 0
	for i.PC < len(i.Image) {
		op := i.Image[i.PC]
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
			if h := i.inH[port]; h != nil {
				i.sp--
				if err = h(i, port); err != nil {
					return err
				}
			} else {
				// we're not calling i.In so that we can optimize out a Pop/Push
				// sequence
				i.data[i.sp], i.Ports[port] = i.Ports[port], 0
			}
			i.PC++
		case OpOut:
			v, port := i.data[i.sp-1], i.data[i.sp]
			i.sp -= 2
			if h := i.outH[port]; h != nil {
				if err = h(i, v, port); err != nil {
					return err
				}
			} else {
				i.Out(v, port)
			}
			i.PC++
		case OpWait:
			if i.Ports[0] != 1 {
				for p, h := range i.waitH {
					v := i.Ports[p]
					if v == 0 {
						continue
					}
					if err = h(i, v, p); err != nil {
						return err
					}
				}
			}
			i.PC++
		default:
			i.rsp++
			i.address[i.rsp], i.PC = Cell(i.PC), int(op)
		}
		i.insCount++
	}
	return nil
}
