//

package vm_test

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/db47h/ngaro/vm"
	"github.com/pkg/errors"
)

type C []vm.Cell

func setup(code, stack, rstack C) *vm.Instance {
	i := vm.New(vm.Image(code), "")
	for _, v := range stack {
		i.Push(v)
	}
	for _, v := range rstack {
		i.Rpush(v)
	}
	return i
}

func check(t *testing.T, i *vm.Instance, ip int, stack C, rstack C) {
	lip, err := i.Run(len(i.Image))
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if ip <= 0 {
		ip = len(i.Image)
	}
	if ip != lip {
		t.Errorf("%+v", errors.Errorf("Bad IP %d != %d", lip, ip))
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
	p := setup(C{7, 100}, C{2}, nil)
	check(t, p, 100, C{1}, nil)

	p = setup(C{7, 100}, C{1}, nil)
	check(t, p, 2, nil, nil)
}

// TODO: make more.. and

type nilOutput struct{}

func (nilOutput) WriteRune(r rune) (size int, err error) { return utf8.RuneLen(r), nil }
func (nilOutput) Flush() error                           { return nil }

func BenchmarkRun(b *testing.B) {
	b.StopTimer()
	//	b.N = 1

	imageFile := "testdata/retroImage"
	input, err := os.Open("testdata/core.rx")
	if err != nil {
		b.Errorf("%+v\n", err)
		return
	}
	defer input.Close()

	for i := 0; i < b.N; i++ {
		img, err := vm.Load(imageFile, 50000)
		if err != nil {
			b.Errorf("%+v\n", err)
			return
		}
		input.Seek(0, 0)
		proc := vm.New(img, imageFile,
			vm.Options.Input(bufio.NewReader(input)),
			vm.Options.Output(nilOutput{}))
		n := time.Now()
		b.StartTimer()
		_, err = proc.Run(len(proc.Image))
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
