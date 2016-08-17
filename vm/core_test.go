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

package vm_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/db47h/ngaro/asm"
	"github.com/db47h/ngaro/vm"
)

type C []vm.Cell

var retroImage = "testdata/retroImage"
var imageBits = 32

func runImage(img []vm.Cell, name string, opts ...vm.Option) (*vm.Instance, error) {
	i, err := vm.New(img, name, opts...)
	if err != nil {
		return nil, err
	}
	return i, i.Run()
}

func runImageFile(name string, bits int, opts ...vm.Option) (*vm.Instance, error) {
	img, _, err := vm.Load(name, 50000, bits)
	if err != nil {
		return nil, err
	}
	return runImage(img, name, opts...)
}

func runAsmImage(assembly, name string, opts ...vm.Option) (*vm.Instance, error) {
	img, err := asm.Assemble(name, strings.NewReader(assembly))
	if err != nil {
		return nil, err
	}
	return runImage(img, name, opts...)
}

func setup(code, stack, rstack C) *vm.Instance {
	i, err := vm.New(code, "")
	if err != nil {
		panic(err)
	}
	for _, v := range stack {
		i.Push(v)
	}
	for _, v := range rstack {
		i.Rpush(v)
	}
	return i
}

func check(t *testing.T, testName string, i *vm.Instance, ip int, stack C, rstack C) bool {
	err := i.Run()
	if err != nil {
		t.Errorf("%+v", err)
		return false
	}
	if ip <= 0 {
		ip = len(i.Mem)
	}
	if ip != i.PC {
		t.Errorf("%v", fmt.Errorf("%s: Bad IP %d != %d", testName, i.PC, ip))
		return false
	}
	stk := i.Data()
	diff := len(stk) != len(stack)
	if !diff {
		for i := range stack {
			if stack[i] != stk[i] {
				diff = true
				break
			}
		}
	}
	if diff {
		t.Errorf("%v", fmt.Errorf("%s: Stack error: expected %d, got %d", testName, stack, stk))
		return false
	}
	stk = i.Address()
	diff = len(stk) != len(rstack)
	if !diff {
		for i := range rstack {
			if rstack[i] != stk[i] {
				diff = true
				break
			}
		}
	}
	if diff {
		t.Errorf("%v", fmt.Errorf("%s: Return stack error: expected %d, got %d", testName, rstack, stk))
		return false
	}
	return true
}

var tests = [...]struct {
	name    string
	code    string
	data    []vm.Cell
	address []vm.Cell
	pc      int
}{
	{"nop", "nop", nil, nil, -1},
	{"lit", "lit 25", C{25}, nil, -1},
	{"dup", "1234 dup", C{1234, 1234}, nil, -1},
	{"drop", "50 drop", nil, nil, -1},
	{"swap", "50 60 swap", C{60, 50}, nil, -1},
	{"push", "82 push", nil, C{82}, -1},
	{"pop", "82 push pop", C{82}, nil, -1},
	{"loop", "3 :REPEAT dup push loop REPEAT", nil, C{3, 2, 1}, -1},
	{"call", "func .org 32 :func 1 2", C{1, 2}, C{0}, -1},
	{"return", "func end .org 32 :func -2 ; :end -1", C{-2, -1}, C{1}, -1},
	{"ZeroExit", `fallthrough return quit
				  .org 32
				  :fallthrough 0 1 0;
				  :return     -1 0 0;
				  :quit`, C{0, 1, -1, -1}, C{2}, -1},
	{"jump", "1 2 jump OVER 3 4 5 :OVER 6 7", C{1, 2, 6, 7}, nil, -1},
	{"<jump", "2 1 <jump END 12 1 2 <jump END 21 :END", C{12}, nil, -1},
	{">jump", "1 2 >jump END 21 2 1 >jump END 12 :END", C{21}, nil, -1},
	{"!jump", "1 1 !jump END 11 1 0 !jump END 10 :END", C{11}, nil, -1},
	{"=jump", "1 0 =jump END 10 1 1 =jump END 11 :END", C{10}, nil, -1},
	{"+", "2 3 +    2 -3 +", C{5, -1}, nil, -1},
	{"-", "2 1 -   1 2 -   1 -2 -   -1 -2 -", C{1, -1, 3, 1}, nil, -1},
	{"*", "0 5 *   1 5 *   5 5 *", C{0, 5, 25}, nil, -1},
	{"/mod", "25 5 /mod  26 5 /mod", C{0, 5, 1, 5}, nil, -1},
	{"1+", "-1 1+   0 1+    1 1+", C{0, 1, 2}, nil, -1},
	{"1-", "-1 1-    0 1-   1 1-", C{-2, -1, 0}, nil, -1},
	{"and", "0 0 and  0 1 and   1 0 and  1 1 and", C{0, 0, 0, 1}, nil, -1},
	{"or", "0 0 or   0 1 or   1 0 or   1 1 or", C{0, 1, 1, 1}, nil, -1},
	{"xor", "0 0 xor   0 1 xor   1 0 xor   1 1 xor   -1 3 xor", C{0, 1, 1, 0, -4}, nil, -1},
	{"<<", "1 1 <<   2 1 <<   3 1 <<   0 2 <<   -1 2 <<  -3 4 <<", C{2, 4, 6, 0, -4, -48}, nil, -1},
	{">>", "2 1 >>   4 1 >>   6 1 >>   0 2 >>   -4 2 >>   -48 4 >>", C{1, 2, 3, 0, -1, -3}, nil, -1},
	{"@", "1234 drop   0 @   1 @", C{1, 1234}, nil, -1},
	{"!", "42 lit foo 1+ ! :foo lit 0", C{42}, nil, -1},
	{"io", "-1 5 out wait 5 in", C{9}, nil, -1},
}

func TestCore(t *testing.T) {
	for _, test := range tests {
		as, err := asm.Assemble(test.name, strings.NewReader(test.code))
		if err != nil {
			t.Error(err)
			continue
		}
		p := setup(as, nil, nil)

		if !check(t, test.name, p, test.pc, test.data, test.address) {
			// disasm
			var b bytes.Buffer
			b.WriteString(test.name)
			b.WriteString(":\n")
			asm.DisassembleAll(as, 0, &b)
			t.Log(b.String())
		}
	}
}

var fib = `
	( loop fib -- n-n )
	push 0 1
	jump 1+
:0	push			( save ctr )
	dup push		( save fib(n-1) )
	+
	pop swap		( stack: fib(n-2) fib(n-1) )
:1	pop
	loop 0-
	swap
	drop
`

var fibRec = `
	( recursive fib )
	fib
	jump end
.org 32
:fib
	dup	1 >jump 0+ ;
:0	1- dup fib swap
	1- fib
	+ ;
:end
`

var fibOpcode = `
	.opcode fib	-1
	fib
`

func fibFunc(v vm.Cell) vm.Cell {
	var v0, v1 vm.Cell = 0, 1
	for v > 1 {
		v0, v1 = v1, v0+v1
		v--
	}
	return v1
}

func fibHandler(i *vm.Instance, opcode vm.Cell) error {
	switch opcode {
	case -1:
		i.Tos = fibFunc(i.Tos)
		return nil
	default:
		return fmt.Errorf("Unsupported opcode value %d", opcode)
	}
}

func Test_Fib_Opcode(t *testing.T) {
	img, err := asm.Assemble("fib-opcode", strings.NewReader(fibOpcode))
	if err != nil {
		t.Fatal(err)
	}
	p := setup(img, C{30}, nil)
	p.SetOptions(vm.BindOpcodeHandler(fibHandler))
	check(t, "Fib-opcode", p, 0, C{832040}, nil)
}

func Test_Fib_AsmLoop(t *testing.T) {
	img, err := asm.Assemble("fib-asm-loop", strings.NewReader(fib))
	if err != nil {
		t.Fatal(err)
	}
	p := setup(img, C{30}, nil)
	check(t, "Fib_AsmLoop", p, 0, C{832040}, nil)
}

func Test_Fib_AsmRecursive(t *testing.T) {
	img, err := asm.Assemble("fib-asm-recursive", strings.NewReader(fibRec))
	if err != nil {
		t.Fatal(err)
	}
	p := setup(img, C{30}, nil)
	check(t, "Fib_AsmRecursive", p, 0, C{832040}, nil)
}

func Test_Fib_RetroLoop(t *testing.T) {
	fib := ": fib [ 0 1 ] dip 1- [ dup [ + ] dip swap ] times swap drop ; 30 fib bye\n"
	i, _ := runImageFile(retroImage, imageBits, vm.Input(strings.NewReader(fib)))
	for c := len(i.Address()); c > 0; c-- {
		i.Rpop()
	}
	check(t, "Fib_RetroLoop", i, 0, C{832040}, nil)
}

func Benchmark_Fib_Opcode(b *testing.B) {
	img, err := asm.Assemble("fib-opcode", strings.NewReader(fibOpcode))
	if err != nil {
		b.Fatal(err)
	}
	i := setup(img, C{}, nil)
	i.SetOptions(vm.BindOpcodeHandler(fibHandler))
	for c := 0; c < b.N; c++ {
		i.PC = 0
		i.Push(35)
		i.Run()
		i.Pop()
	}
}

func Benchmark_Fib_AsmLoop(b *testing.B) {
	img, err := asm.Assemble("fib-asm-loop", strings.NewReader(fib))
	if err != nil {
		b.Fatal(err)
	}
	i := setup(img, C{35}, nil)
	for c := 0; c < b.N; c++ {
		i.PC = 0
		i.Run()
		i.Pop()
		i.Push(35)
	}
}

func Benchmark_Fib_AsmRecursive(b *testing.B) {
	img, err := asm.Assemble("fib-asm-recursive", strings.NewReader(fibRec))
	if err != nil {
		b.Fatal(err)
	}
	i := setup(img, C{35}, nil)
	for c := 0; c < b.N; c++ {
		i.PC = 0
		i.Run()
		i.Pop()
		i.Push(35)
	}
}

func Benchmark_Fib_RetroLoop(b *testing.B) {
	fib := ": fib [ 0 1 ] dip 1- [ dup [ + ] dip swap ] times swap drop ; 35 fib bye\n"
	for c := 0; c < b.N; c++ {
		b.StopTimer()
		img, _, _ := vm.Load(retroImage, 50000, imageBits)
		i, _ := vm.New(img, retroImage,
			vm.Input(strings.NewReader(fib)))
		b.StartTimer()
		i.Run()
	}
}

func Benchmark_Fib_RetroRecursive(b *testing.B) {
	fib := ": fib dup 2 < if; 1- dup fib swap 1- fib + ; 35 fib bye\n"
	for c := 0; c < b.N; c++ {
		b.StopTimer()
		img, _, _ := vm.Load(retroImage, 50000, imageBits)
		i, _ := vm.New(img, retroImage,
			vm.Input(strings.NewReader(fib)))
		b.StartTimer()
		i.Run()
	}
}

func assertEqual(t *testing.T, name, expected, got string) {
	if expected != got {
		t.Errorf("%v:\nExpected: %v\nGot: %v", name, expected, got)
	}
}
func assertEqualI(t *testing.T, name string, expected, got int) {
	if expected != got {
		t.Errorf("%v:\nExpected: %v\nGot: %v", name, expected, got)
	}
}

func TestVM_Nos(t *testing.T) {
	test := "testNOS"
	i, err := vm.New(nil, "")
	if err != nil {
		panic(err)
	}
	assertEqualI(t, test, 0, int(i.Nos()))
	i.Push(4)
	assertEqualI(t, test, 0, int(i.Nos()))
	i.Push(3)
	assertEqualI(t, test, 4, int(i.Nos()))
}

func TestVM_Drop2(t *testing.T) {
	test := "testDrop2"
	i, err := vm.New(nil, "")
	if err != nil {
		panic(err)
	}
	assertEqualI(t, test, 0, int(i.Depth()))
	i.Drop2()
	assertEqualI(t, test, 0, int(i.Depth()))
	assertEqualI(t, test, 0, int(i.Tos))
	assertEqualI(t, test, 0, int(i.Nos()))
	i.Push(4)
	i.Drop2()
	assertEqualI(t, test, 0, int(i.Depth()))
	assertEqualI(t, test, 0, int(i.Tos))
	assertEqualI(t, test, 0, int(i.Nos()))
	i.Push(4)
	i.Push(7)
	i.Drop2()
	assertEqualI(t, test, 0, int(i.Depth()))
	assertEqualI(t, test, 0, int(i.Tos))
	assertEqualI(t, test, 0, int(i.Nos()))
	i.Push(4)
	i.Push(7)
	i.Push(8)
	i.Drop2()
	assertEqualI(t, test, 1, int(i.Depth()))
	assertEqualI(t, test, 4, int(i.Tos))
	assertEqualI(t, test, 0, int(i.Nos()))
}

func TestVM_error(t *testing.T) {
	_, err := runAsmImage("16 @", "VM_error")
	if err == nil {
		t.Fatal("Unexpected nil error")
	}
	assertEqual(t, "VM_error", "Recovered error @pc=2/3, stack 1/1024, rstack 0/1024: runtime error: index out of range", err.Error())
}

func TestVM_inHandler(t *testing.T) {
	i, err := runAsmImage("43 in", "VM_inHandler",
		vm.BindInHandler(43, func(i *vm.Instance, p vm.Cell) error {
			if p != 43 {
				return fmt.Errorf("Wrong port number %d", p)
			}
			i.Push(42)
			return nil
		}))
	if err != nil {
		t.Fatal(err)
	}
	assertEqualI(t, "VM_inHandler", 42, int(i.Tos))
}

func TestVM_InstructionCount(t *testing.T) {
	i, err := runAsmImage("10 :0 loop 0-", "VM_InstructionCount")
	if err != nil {
		t.Fatal(err)
	}
	assertEqualI(t, "VM_InstructionCount", 11, int(i.InstructionCount()))
}

func TestVM_DataSize(t *testing.T) {
	i, err := vm.New(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 10; n++ {
		i.Push(vm.Cell(n))
	}
	err = i.SetOptions(vm.DataSize(9))
	if err == nil {
		t.Fatal("Unexpected nil error")
	}
	err = i.SetOptions(vm.DataSize(10))
	if err != nil {
		t.Fatal(err)
	}
	assertEqualI(t, "VM_DataSize", 10, len(i.Data()))
}

func TestVM_AddressSize(t *testing.T) {
	i, err := vm.New(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 10; n++ {
		i.Rpush(vm.Cell(n))
	}
	err = i.SetOptions(vm.AddressSize(9))
	if err == nil {
		t.Fatal("Unexpected nil error")
	}
	err = i.SetOptions(vm.AddressSize(10))
	if err != nil {
		t.Fatal(err)
	}
	assertEqualI(t, "VM_DataSize", 10, len(i.Address()))
}
