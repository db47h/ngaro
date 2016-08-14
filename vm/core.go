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

// Depth returns the data stack depth.
func (i *Instance) Depth() int {
	return i.sp - 1
}

// Drop removes the top item from the data stack.
func (i *Instance) Drop() {
	if i.sp > 1 {
		i.sp--
		i.Tos = i.data[i.sp]
	} else {
		i.Tos = 0
	}
}

// Drop2 removes the top two items from the data stack.
func (i *Instance) Drop2() {
	i.sp -= 2
	if i.sp >= 1 {
		i.Tos = i.data[i.sp]
	} else {
		i.sp = 1
		i.Tos = 0
	}
}

// Push pushes the argument on top of the data stack.
func (i *Instance) Push(v Cell) {
	i.data[i.sp], i.Tos = i.Tos, v
	i.sp++
}

// Pop pops the value on top of the data stack and returns it.
func (i *Instance) Pop() Cell {
	tos := i.Tos
	if i.sp > 1 {
		i.sp--
		i.Tos = i.data[i.sp]
	} else {
		i.Tos = 0
	}
	return tos
}

// Rpush pushes the argument on top of the address stack.
func (i *Instance) Rpush(v Cell) {
	i.address[i.rsp], i.rtos = i.rtos, v
	i.rsp++
}

// Rpop pops the value on top of the address stack and returns it.
func (i *Instance) Rpop() Cell {
	rtos := i.rtos
	if i.rsp > 1 {
		i.rsp--
		i.rtos = i.address[i.rsp]
	} else {
		i.rtos = 0
	}
	return rtos
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
				err = errors.Wrapf(e, "Recovered error @pc=%d/%d, stack %d/%d, rstack %d/%d", i.PC, len(i.Image), i.sp, len(i.data), i.rsp, len(i.address))
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
			i.data[i.sp] = i.Tos
			i.sp++
			i.PC++
		case OpDrop:
			i.Drop()
			i.PC++
		case OpSwap:
			i.Tos, i.data[i.sp-1] = i.data[i.sp-1], i.Tos
			i.PC++
		case OpPush:
			i.Rpush(i.Pop())
			i.PC++
		case OpPop:
			i.Push(i.Rpop())
			i.PC++
		case OpLoop:
			v := i.Tos - 1
			if v > 0 {
				i.Tos = v
				i.PC = int(i.Image[i.PC+1])
			} else {
				i.Drop()
				i.PC += 2
			}
		case OpJump:
			i.PC = int(i.Image[i.PC+1])
		case OpReturn:
			i.PC = int(i.Rpop() + 1)
		case OpGtJump:
			if i.data[i.sp-1] > i.Tos {
				i.PC = int(i.Image[i.PC+1])
			} else {
				i.PC += 2
			}
			i.Drop2()
		case OpLtJump:
			if i.data[i.sp-1] < i.Tos {
				i.PC = int(i.Image[i.PC+1])
			} else {
				i.PC += 2
			}
			i.Drop2()
		case OpNeJump:
			if i.data[i.sp-1] != i.Tos {
				i.PC = int(i.Image[i.PC+1])
			} else {
				i.PC += 2
			}
			i.Drop2()
		case OpEqJump:
			if i.data[i.sp-1] == i.Tos {
				i.PC = int(i.Image[i.PC+1])
			} else {
				i.PC += 2
			}
			i.Drop2()
		case OpFetch:
			i.Tos = i.Image[i.Tos]
			i.PC++
		case OpStore:
			i.Image[i.Tos] = i.data[i.sp-1]
			i.Drop2()
			i.PC++
		case OpAdd:
			rhs := i.Pop()
			i.Tos += rhs
			i.PC++
		case OpSub:
			rhs := i.Pop()
			i.Tos -= rhs
			i.PC++
		case OpMul:
			rhs := i.Pop()
			i.Tos *= rhs
			i.PC++
		case OpDimod:
			lhs, rhs := i.data[i.sp-1], i.Tos
			i.data[i.sp-1] = lhs % rhs
			i.Tos = lhs / rhs
			i.PC++
		case OpAnd:
			rhs := i.Pop()
			i.Tos &= rhs
			i.PC++
		case OpOr:
			rhs := i.Pop()
			i.Tos |= rhs
			i.PC++
		case OpXor:
			rhs := i.Pop()
			i.Tos ^= rhs
			i.PC++
		case OpShl:
			rhs := i.Pop()
			i.Tos <<= uint8(rhs)
			i.PC++
		case OpShr:
			rhs := i.Pop()
			i.Tos >>= uint8(rhs)
			i.PC++
		case OpZeroExit:
			if i.Tos == 0 {
				i.PC = int(i.Rpop() + 1)
				i.Drop()
			} else {
				i.PC++
			}
		case OpInc:
			i.Tos++
			i.PC++
		case OpDec:
			i.Tos--
			i.PC++
		case OpIn:
			port := i.Tos
			if h := i.inH[port]; h != nil {
				i.Drop()
				if err = h(i, port); err != nil {
					return err
				}
			} else {
				// we're not calling i.In so that we can optimize out a Pop/Push
				// sequence
				i.Tos, i.Ports[port] = i.Ports[port], 0
			}
			i.PC++
		case OpOut:
			v, port := i.data[i.sp-1], i.Tos
			i.Drop2()
			if h := i.outH[port]; h != nil {
				err = h(i, v, port)
			} else {
				err = i.Out(v, port)
			}
			if err != nil {
				return err
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
			i.address[i.rsp] = i.rtos
			i.rsp++
			i.rtos, i.PC = Cell(i.PC), int(op)
		}
		i.insCount++
	}
	return nil
}
