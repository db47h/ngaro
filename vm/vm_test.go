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
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/db47h/ngaro/asm"
	"github.com/db47h/ngaro/vm"
)

type C []vm.Cell

var imageFile = "testdata/retroImage"

func setup(code, stack, rstack C) *vm.Instance {
	i, err := vm.New(vm.Image(code), "")
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

func check(t *testing.T, testName string, i *vm.Instance, ip int, stack C, rstack C) {
	err := i.Run()
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if ip <= 0 {
		ip = len(i.Image)
	}
	if ip != i.PC {
		t.Errorf("%v", fmt.Errorf("%s: Bad IP %d != %d", testName, i.PC, ip))
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
	}
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
	{"call", "sub .org 32 :sub 1 2", C{1, 2}, C{0}, -1},
	{"return", "sub end .org 32 :sub -2 ; :end -1", C{-2, -1}, C{1}, -1},
	{"ZeroExit", `fallthrough return quit
				  .org 32
				  :fallthrough 0 1 0;
				  :return     -1 0 0;
				  :quit nop`, C{0, 1, -1, -1}, C{2}, -1},
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
		check(t, test.name, p, test.pc, test.data, test.address)
		if t.Failed() {
			// disasm
			var b bytes.Buffer
			b.WriteString(test.name)
			b.WriteString(":\n")
			for pc := 0; pc < len(as); {
				fmt.Fprintf(&b, "% 4d\t", pc)
				pc = asm.Disassemble(as, pc, &b)
				b.WriteByte('\n')
			}
			t.Log(b.String())
		}
	}
}

// TODO: make more...

var fib = []vm.Cell{
	vm.OpPush,
	vm.OpLit, 0,
	vm.OpLit, 1,
	vm.OpPop,
	vm.OpJump, 15, // junp to loop
	vm.OpPush, // save count
	vm.OpDup,
	vm.OpPush, // save n-1
	vm.OpAdd,  // n-2 + n-1
	vm.OpPop,
	vm.OpSwap, // stack: n-2 n-1
	vm.OpPop,
	vm.OpLoop, 8, // loop
	vm.OpSwap,
	vm.OpDrop,
}

var nFib = []vm.Cell{
	vm.OpJump, 32,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	44, // call
	vm.OpLit, -9,
	vm.OpLit, 5,
	vm.OpOut,
	vm.OpLit, 0,
	vm.OpLit, 0,
	vm.OpOut,
	vm.OpWait,
	// entry
	vm.OpDup,
	vm.OpLit, 1,
	vm.OpGtJump, 50,
	vm.OpReturn,
	vm.OpDec,
	vm.OpDup,
	44,
	vm.OpSwap,
	vm.OpDec,
	44,
	vm.OpAdd,
	vm.OpReturn,
}

func Test_Fib_AsmLoop(t *testing.T) {
	p := setup(fib, C{30}, nil)
	check(t, "Fib_AsmLoop", p, 0, C{832040}, nil)
}

func Test_Fib_AsmRecursive(t *testing.T) {
	p := setup(nFib, C{30}, nil)
	check(t, "Fib_AsmRecursive", p, 0, C{832040}, nil)
}

func Test_Fib_RetroLoop(t *testing.T) {
	fib := ": fib [ 0 1 ] dip 1- [ dup [ + ] dip swap ] times swap drop ; 30 fib bye\n"
	img, _, _ := vm.Load(imageFile, 50000)
	i, _ := vm.New(img, imageFile,
		vm.Input(strings.NewReader(fib)))
	i.Run()
	for c := len(i.Address()); c > 0; c-- {
		i.Rpop()
	}
	check(t, "Fib_RetroLoop", i, 0, C{832040}, nil)
}

func Benchmark_Fib_AsmLoop(b *testing.B) {
	f := make(C, len(fib))
	for i := range fib {
		f[i] = vm.Cell(fib[i])
	}
	i := setup(f, C{35}, nil)
	for c := 0; c < b.N; c++ {
		i.PC = 0
		i.Run()
		i.Pop()
		i.Push(35)
	}
}

func Benchmark_Fib_AsmRecursive(b *testing.B) {
	f := make(C, len(nFib))
	for i := range nFib {
		f[i] = vm.Cell(nFib[i])
	}
	i := setup(f, C{35}, nil)
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
		img, _, _ := vm.Load(imageFile, 50000)
		i, _ := vm.New(img, imageFile,
			vm.Input(strings.NewReader(fib)))
		b.StartTimer()
		i.Run()
	}
}

func Benchmark_Fib_RetroRecursive(b *testing.B) {
	fib := ": fib dup 2 < if; 1- dup fib swap 1- fib + ; 35 fib bye\n"
	for c := 0; c < b.N; c++ {
		b.StopTimer()
		img, _, _ := vm.Load(imageFile, 50000)
		i, _ := vm.New(img, imageFile,
			vm.Input(strings.NewReader(fib)))
		b.StartTimer()
		i.Run()
	}
}

func BenchmarkRun(b *testing.B) {
	input, err := os.Open("testdata/core.rx")
	if err != nil {
		b.Errorf("%+v\n", err)
		return
	}
	defer input.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		img, _, err := vm.Load(imageFile, 50000)
		if err != nil {
			b.Fatalf("%+v\n", err)
		}
		input.Seek(0, 0)
		proc, err := vm.New(img, imageFile, vm.Input(input))
		if err != nil {
			panic(err)
		}

		n := time.Now()
		b.StartTimer()

		err = proc.Run()

		b.StopTimer()
		el := time.Now().Sub(n).Seconds()
		c := proc.InstructionCount()

		fmt.Printf("Executed %d instructions in %.3fs. Perf: %.2f MIPS\n", c, el, float64(c)/1e6/el)
		if err != nil {
			switch err {
			case io.EOF: // stdin or stdout closed
			default:
				b.Errorf("%+v\n", err)
			}
		}
	}
}
