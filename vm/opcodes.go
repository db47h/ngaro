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

var opcodes = [...]string{
	"nop",
	"",
	"dup",
	"drop",
	"swap",
	"push",
	"pop",
	"loop",
	"jump",
	";",
	">jump",
	"<jump",
	"!jump",
	"=jump",
	"@",
	"!",
	"+",
	"-",
	"*",
	"/mod",
	"and",
	"or",
	"xor",
	"<<",
	">>",
	"0;",
	"1+",
	"1-",
	"in",
	"out",
	"wait",
}

var opcodeIndex = make(map[string]Cell)

func init() {
	for i, v := range opcodes {
		opcodeIndex[v] = Cell(i)
	}
}
