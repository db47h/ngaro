//

package vm_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/db47h/ngaro/vm"
	"github.com/pkg/errors"
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

func check(t *testing.T, i *vm.Instance, ip int, stack C, rstack C) {
	err := i.Run(len(i.Image))
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if ip <= 0 {
		ip = len(i.Image)
	}
	if ip != i.PC {
		t.Errorf("%+v", errors.Errorf("Bad IP %d != %d", i.PC, ip))
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
		t.Errorf("%+v", errors.Errorf("Stack error: expected %d, got %d", stack, stk))
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
		t.Errorf("%+v", errors.Errorf("Return stack error: expected %d, got %d", rstack, stk))
	}
}

func TestLit(t *testing.T) {
	p := setup(C{1, 25}, nil, nil)
	check(t, p, 0, C{25}, nil)
}

func TestDup(t *testing.T) {
	p := setup(C{2}, C{0, 42}, nil)
	check(t, p, 0, C{0, 42, 42}, nil)
}

func TestDrop(t *testing.T) {
	p := setup(C{3}, C{0, 42}, nil)
	check(t, p, 0, C{0}, nil)
}

func TestSwap(t *testing.T) {
	p := setup(C{4}, C{0, 42}, nil)
	check(t, p, 0, C{42, 0}, nil)
}

func TestPush(t *testing.T) {
	p := setup(C{5}, C{42}, nil)
	check(t, p, 0, nil, C{42})
}

func TestPop(t *testing.T) {
	p := setup(C{6}, nil, C{42})
	check(t, p, 0, C{42}, nil)
}

func TestLoop(t *testing.T) {
	p := setup(C{
		7, 4, // 0: LOOP 4
		1, 0, // 2: LIT 0
		1, 1, // 4: LIT 1
	}, C{43}, nil)
	check(t, p, 0, C{42, 1}, nil)

	p = setup(C{7, 4, 1, 0, 1, 1}, C{1}, nil)
	check(t, p, 0, C{0, 1}, nil)
}

// TODO: make more...

// Shows how to load an image, setup the VM with multiple readers/init code.
func ExampleInstance_Run() {
	imageFile := "testdata/retroImage"
	img, err := vm.Load(imageFile, 50000)
	if err != nil {
		panic(err)
	}

	// output capture buffer
	output := bytes.NewBuffer(nil)

	// Setup the VM instance with os.Stdin as first reader, and we push another
	// reader with some custom init code that will include and run the retro core tests.
	i, err := vm.New(img, imageFile,
		vm.Input(os.Stdin),
		vm.Input(strings.NewReader("\"testdata/core.rx\" :include\n")),
		vm.Output(output))

	// run it
	if err == nil {
		err = i.Run(len(i.Image))
	}
	if err != nil {
		// in interactive use, err may be io.EOF if any of the IO channels gets closed
		// in which case this would be a normal exit condition
		panic(err)
	}

	// filter output to get the retro core test results.
	b := bytes.Split(output.Bytes(), []byte{'\n'})
	fmt.Printf("%s\n", b[len(b)-5])
	fmt.Printf("%s\n", b[len(b)-4])

	// Output:
	// 360 tests run: 360 passed, 0 failed.
	// 186 words checked, 0 words unchecked, 37 i/o words ignored.
}

// Shows how to setup a port handler.
func ExampleOutHandler() {
	imageFile := "testdata/retroImage"
	img, err := vm.Load(imageFile, 0)
	if err != nil {
		panic(err)
	}

	// we will use a buffered stdout
	output := bufio.NewWriter(os.Stdout)
	// so according to the spec, we should flush the output as soon as port 3
	// is written to:
	outputHandler := func(v vm.Cell) (vm.Cell, error) {
		output.Flush()
		return 0, nil
	}

	i, err := vm.New(img, imageFile,
		vm.Input(strings.NewReader("6 7 * putn\n")),
		vm.Output(output),
		vm.OutHandler(3, outputHandler))
	if err != nil {
		panic(err)
	}

	if err = i.Run(len(i.Image)); err != nil && errors.Cause(err) != io.EOF {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}
	// Retro 11.7.1
	//
	// ok  6
	// ok  7
	// ok  *
	// ok  putn 42
	// ok
}

var fib = []vm.Opcode{
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

var nFib = []vm.Opcode{
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
	f := make(C, len(fib))
	for i := range fib {
		f[i] = vm.Cell(fib[i])
	}
	p := setup(f, C{30}, nil)
	check(t, p, 0, C{832040}, nil)
}

func Test_Fib_AsmRecursive(t *testing.T) {
	f := make(C, len(nFib))
	for i := range nFib {
		f[i] = vm.Cell(nFib[i])
	}
	p := setup(f, C{30}, nil)
	check(t, p, 0, C{832040}, nil)
}

func Test_Fib_RetroLoop(t *testing.T) {
	fib := ": fib [ 0 1 ] dip 1- [ dup [ + ] dip swap ] times swap drop ; 30 fib bye\n"
	img, _ := vm.Load(imageFile, 50000)
	i, _ := vm.New(img, imageFile,
		vm.Input(strings.NewReader(fib)))
	i.Run(len(i.Image))
	for c := len(i.Address()); c > 0; c-- {
		i.Rpop()
	}
	check(t, i, 0, C{832040}, nil)
}

func Benchmark_Fib_AsmLoop(b *testing.B) {
	f := make(C, len(fib))
	for i := range fib {
		f[i] = vm.Cell(fib[i])
	}
	i := setup(f, C{35}, nil)
	for c := 0; c < b.N; c++ {
		i.PC = 0
		i.Run(len(i.Image))
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
		i.Run(len(i.Image))
		i.Pop()
		i.Push(35)
	}
}

func Benchmark_Fib_RetroLoop(b *testing.B) {
	fib := ": fib [ 0 1 ] dip 1- [ dup [ + ] dip swap ] times swap drop ; 35 fib bye\n"
	for c := 0; c < b.N; c++ {
		b.StopTimer()
		img, _ := vm.Load(imageFile, 50000)
		i, _ := vm.New(img, imageFile,
			vm.Input(strings.NewReader(fib)))
		b.StartTimer()
		i.Run(len(i.Image))
	}
}

func Benchmark_Fib_RetroRecursive(b *testing.B) {
	fib := ": fib dup 2 < if; 1- dup fib swap 1- fib + ; 35 fib bye\n"
	for c := 0; c < b.N; c++ {
		b.StopTimer()
		img, _ := vm.Load(imageFile, 50000)
		i, _ := vm.New(img, imageFile,
			vm.Input(strings.NewReader(fib)))
		b.StartTimer()
		i.Run(len(i.Image))
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
		img, err := vm.Load(imageFile, 50000)
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

		err = proc.Run(len(proc.Image))

		b.StopTimer()
		el := time.Now().Sub(n).Seconds()
		c := proc.InstructionCount()

		fmt.Printf("Executed %d instructions in %.3fs. Perf: %.2f MIPS\n", c, el, float64(c)/1e6/el)
		if err != nil {
			switch errors.Cause(err) {
			case io.EOF: // stdin or stdout closed
			default:
				b.Errorf("%+v\n", err)
			}
		}
	}
}
