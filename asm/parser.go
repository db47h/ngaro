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

package asm

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"
	"unsafe"

	"github.com/db47h/ngaro/vm"
)

const localSep = "·"
const maxErrors = 10

// ErrAsm encapsulates errors generated by the assembler.
type ErrAsm []struct {
	Pos scanner.Position
	Msg string
}

func (e ErrAsm) Error() string {
	l := make([]string, 0, len(e))
	for _, err := range e {
		l = append(l, fmt.Sprintf("%s: %s", err.Pos, err.Msg))
	}
	return strings.Join(l, "\n")
}

// labelSite registers at witch address and position in the source stream a
// given label is used.
type labelSite struct {
	pos     scanner.Position
	address int
}

// label keeps track of all uses of a given label.
type label struct {
	labelSite             // where the label is defined
	uses      []labelSite // where it's used
}

// parser provides the parsing and compiling.
type parser struct {
	i       []vm.Cell
	pc      int
	s       scanner.Scanner
	labels  map[string]*label
	locCtr  map[int]int
	consts  map[string]labelSite
	cstName string
	cstPos  scanner.Position
	errs    ErrAsm
}

func newParser() *parser {
	p := new(parser)
	p.labels = make(map[string]*label)
	p.locCtr = make(map[int]int)
	p.consts = make(map[string]labelSite)
	return p
}

// helper to build ErrAsm items.
func parseError(pos scanner.Position, msg string) struct {
	Pos scanner.Position
	Msg string
} {
	return struct {
		Pos scanner.Position
		Msg string
	}{pos, msg}
}

// Error appends an error to the internal error list at the current scanner pos.
func (p *parser) error(msg string) {
	pos := p.s.Position
	if !pos.IsValid() {
		pos = p.s.Pos()
	}
	p.errs = append(p.errs, parseError(pos, msg))
}

// abort returns true if the parser should abort due to too many errors.
func (p *parser) abort() bool { return len(p.errs) >= maxErrors }

// write is the actual compilation function. It emits the given value at the
// current compile address, then imcrements it. It also takes care of managing
// the image size.
func (p *parser) write(v vm.Cell) {
	for p.pc >= len(p.i) {
		p.i = append(p.i, make([]vm.Cell, 16384)...)
	}
	p.i[p.pc] = v
	p.pc++
}

// isLocalLabel checks whether a label is local (i.e. numeric).
func isLocalLabel(name string) (int, bool) {
	n, err := strconv.Atoi(name)
	return n, err == nil
}

// makeLabelRef registers the use of the given label at the current position.
func (p *parser) makeLabelRef(name string) {
	var (
		isLocal bool
		look    byte
		n       int
		lbl     *label
		pos     = p.s.Position
	)

	// demangle name and check if local
	if len(name) > 1 {
		if look = name[len(name)-1]; look == '-' || look == '+' {
			t := name[:len(name)-1]
			n, isLocal = isLocalLabel(t)
			if isLocal {
				name = t
			}
		}
	}

	switch isLocal {
	case true:
		switch look {
		case '-':
			// build name
			t := name + localSep + strconv.Itoa(p.locCtr[n]) // last index suffix
			lbl = p.labels[t]
			if lbl == nil {
				p.error("Backward reference to undefined local label " + name)
				return
			}
		case '+':
			// build name
			t := name + localSep + strconv.Itoa(p.locCtr[n]+1) // next index suffix
			lbl = p.labels[t]
			if lbl == nil {
				lbl = &label{
					labelSite{pos, -1},
					nil,
				}
				p.labels[t] = lbl
			}
		}
	case false:
		lbl = p.labels[name]
		if lbl == nil {
			lbl = &label{
				// use current position as valid temp position
				labelSite{pos, -1},
				nil,
			}
			p.labels[name] = lbl
		}
	}
	lbl.uses = append(lbl.uses, labelSite{pos, p.pc})
}

func isIdentRune(ch rune, i int) bool {
	return unicode.IsLetter(ch) || unicode.IsSymbol(ch) || unicode.IsPunct(ch) || unicode.IsDigit(ch)
}

// Parse does the parsing and compiling. Returns the compiled VM image as a Cell
// slice and any error that occured. If not nil, the returned error can safely
// be cast to an ErrAsm value that will contain up to 10 entries.
func (p *parser) Parse(name string, r io.Reader) ([]vm.Cell, error) {
	// state:
	// 0: accept anything
	// 1: need integer, const or address argument (lit, loop and jumps)
	// 2: accept integer or const (for .org directive)
	// 3: accept integer or const (for .equ value)
	var state int

	p.s.Init(r)
	p.s.Error = func(s *scanner.Scanner, msg string) {
		pos := s.Position
		if !pos.IsValid() {
			pos = s.Pos()
		}
		p.errs = append(p.errs, parseError(pos, msg))
	}
	p.s.IsIdentRune = isIdentRune
	p.s.Mode = scanner.ScanIdents
	p.s.Filename = name

	for tok := p.s.Scan(); !p.abort() && tok != scanner.EOF; tok = p.s.Scan() {
		var v int
		s := p.s.TokenText()

		// Our assembly is forth like: words can start with and contain digits,
		// symbols, punctuation and so on. The stdlib scanner can only return
		// tokens, so we need to convert back to Ints when required.
		// Chars are only a special case of ints.
		switch tok {
		case scanner.Ident:
			// check int
			n, err := strconv.ParseInt(s, 0, 8*int(unsafe.Sizeof(vm.Cell(0))))
			if err == nil {
				tok = scanner.Int
				v = int(n)
				break
			}
			// check char
			if len(s) > 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
				r, _, _, err := strconv.UnquoteChar(s[1:len(s)-1], '\'')
				if err != nil {
					p.error(err.Error())
					break
				}
				v = int(r)
				tok = scanner.Int
				break
			}
			// constant ?
			c, ok := p.consts[s]
			if ok {
				v = c.address
				tok = scanner.Int
				break
			}
		default:
			p.error("Unexpected character " + strconv.QuoteRune(tok))
		}

	s: // now we only have ints or idents
		switch tok {
		case scanner.Int:
			switch state {
			case 2:
				// .org
				p.pc = v
			case 3:
				// .equ
				p.consts[p.cstName] = labelSite{p.cstPos, v}
			case 0:
				// implicit lit
				p.write(vm.OpLit)
				fallthrough
			default: // (1)
				// argument
				p.write(vm.Cell(v))
			}
			state = 0
		case scanner.Ident:
			switch s[0] {
			case ':':
				if state != 0 {
					p.error("Unexpected label definition as argument: " + s)
					// attempt to continue parsing with state = 0
					state = 0
				}
				n := s[1:]
				if len(n) == 0 {
					p.error("Empty label name")
					break s
				}
				if cst, ok := p.consts[n]; ok {
					p.error("Label redefinition:" + n + ", prefiously defined as a constant here:" + cst.pos.String())
					break s
				}
				// local label?
				if i, ok := isLocalLabel(n); ok {
					// increment counter and update name
					idx := p.locCtr[i] + 1
					p.locCtr[i] = idx
					n = n + localSep + strconv.Itoa(idx)
				}
				if l, ok := p.labels[n]; ok {
					// set address of forward declaration
					if l.address != -1 {
						p.error("Label redefinition: " + n + ", previous definition here:" + l.pos.String())
					}
					l.address = p.pc
					l.pos = p.s.Position
				} else {
					// new label
					p.labels[n] = &label{
						labelSite{p.s.Position, p.pc},
						nil,
					}
				}
			case '.':
				if state != 0 {
					p.error("Unexpected directive as argument: " + s)
					// attempt to keep parsing as if state = 0
					state = 0
				}
				switch s {
				case ".org":
					state = 2
				case ".dat":
					state = 1
				case ".equ":
					t := p.s.Scan()
					if t != scanner.Ident {
						p.error(".equ: expected identifier, got " + p.s.TokenText())
						// just eat up next token and keep parsing
						p.s.Scan()
						break s
					}
					p.cstName = p.s.TokenText()
					if l, ok := p.labels[p.cstName]; ok {
						p.error(".equ: redifinition of " + p.cstName + ", previously defined/used as a label: here: " + l.pos.String())
						// just eat up next token and keep parsing
						p.s.Scan()
						break s
					}
					p.cstPos = p.s.Position
					state = 3
				default:
					p.error("Unknown dot directive: " + s)
				}
			default:
				if s == "(" {
					// skip comments
					for ; !p.abort() && tok != scanner.EOF && (tok != scanner.Ident || p.s.TokenText() != ")"); tok = p.s.Scan() {
					}
					break s
				}
				if state >= 2 {
					p.error("Unexpected label as directive argument: " + s)
					// attempt to continue parsing with state = 0
					state = 0
					break s
				}
				if op, ok := opcodeIndex[s]; state == 0 && ok {
					p.write(op)
					switch op {
					case vm.OpLit, vm.OpLoop, vm.OpJump, vm.OpGtJump, vm.OpLtJump, vm.OpNeJump, vm.OpEqJump:
						state = 1
					}
				} else {
					// handle the case of implicit call at pc <= 30
					if state == 0 && p.pc < 31 {
						p.write(vm.OpLit)
						p.write(vm.Cell(p.pc + 3))
						p.write(vm.OpPush)
						p.write(vm.OpJump)
					}
					p.makeLabelRef(s)
					p.write(0)
					state = 0
				}
			}
		}
	}

	// write labels
l:
	for n, l := range p.labels {
		for _, u := range l.uses {
			if l.address == -1 {
				p.errs = append(p.errs, parseError(u.pos, "Undefined label "+n))
				if p.abort() {
					break l
				}
			}
			p.i[u.address] = vm.Cell(l.address)
		}
	}

	if len(p.errs) > 0 {
		return nil, p.errs
	}
	return p.i[:p.pc], nil
}
