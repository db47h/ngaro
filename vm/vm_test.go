//

package vm

import (
	"testing"

	"github.com/pkg/errors"
)

type C []Cell

func setup(code, stack, rstack C) *Instance {
	p := New(Image(code), "")
	p.data = append(stack, make(C, 100)...)
	p.sp = len(stack) - 1
	p.address = append(rstack, make(C, 100)...)
	p.rsp = len(rstack) - 1
	return p
}

func check(t *testing.T, p *Instance, ip int, stack C, rstack C) {
	p.Run(len(p.Image))
	if ip <= 0 {
		ip = len(p.Image)
	}
	if ip != p.ip {
		t.Errorf("%+v", errors.Errorf("Bad IP %d != %d", p.ip, ip))
	}
	diff := (p.sp + 1) != len(stack)
	if !diff {
		for i := range stack {
			if stack[i] != p.data[i] {
				diff = true
				break
			}
		}
	}
	if diff {
		t.Errorf("%+v", errors.Errorf("Stack error: expected %d, got %d", stack, p.data[:p.sp+1]))
	}
	diff = (p.rsp + 1) != len(rstack)
	if !diff {
		for i := range rstack {
			if rstack[i] != p.address[i] {
				diff = true
				break
			}
		}
	}
	if diff {
		t.Errorf("%+v", errors.Errorf("Return stack error: expected %d, got %d", rstack, p.address[:p.rsp+1]))
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
