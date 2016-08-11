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
	"text/scanner"
	"unicode"
	"unsafe"

	"github.com/db47h/ngaro/vm"
)

func isIdentRune(ch rune, i int) bool {
	return unicode.IsLetter(ch) || unicode.IsSymbol(ch) || unicode.IsPunct(ch) || unicode.IsDigit(ch)
}

type labelSite struct {
	pos     scanner.Position
	address int
}

type label struct {
	labelSite
	uses []labelSite
}

type parser struct {
	i       []vm.Cell
	pc      int
	s       scanner.Scanner
	labels  map[string]*label
	consts  map[string]labelSite
	cstName string
	cstPos  scanner.Position
	err     error
}

func newParser() *parser {
	p := new(parser)
	p.labels = make(map[string]*label)
	p.consts = make(map[string]labelSite)
	return p
}

func (p *parser) write(v vm.Cell) {
	for p.pc >= len(p.i) {
		p.i = append(p.i, make([]vm.Cell, 16384)...)
	}
	p.i[p.pc] = v
	p.pc++
}

func (p *parser) useLabel(name string) error {
	lbl := p.labels[name]
	if lbl == nil {
		lbl = &label{
			// use current position as valid temp position
			labelSite{p.s.Pos(), -1},
			nil,
		}
		p.labels[name] = lbl
	}
	lbl.uses = append(lbl.uses, labelSite{p.s.Pos(), p.pc})
	return nil
}

func scanError(s *scanner.Scanner, msg string) error {
	pos := s.Position
	if !pos.IsValid() {
		pos = s.Pos()
	}
	return fmt.Errorf("%s: %s\n", pos, msg)
}

// Parse does the parsing and compiling.
func (p *parser) Parse(name string, r io.Reader) error {
	// state:
	// 0: accept anything
	// 1: need integer, const or address argument (lit, loop and jumps)
	// 2: accept integer or const (for .org directive)
	// 3: accept integer or const (for .equ value)
	var state int

	p.s.Init(r)
	p.s.Error = func(s *scanner.Scanner, msg string) {
		p.err = scanError(s, msg)
	}
	p.s.IsIdentRune = isIdentRune
	p.s.Mode = scanner.ScanIdents
	p.s.Filename = name

	for tok := p.s.Scan(); p.err == nil && tok != scanner.EOF; tok = p.s.Scan() {
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
					p.err = scanError(&p.s, err.Error())
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
			p.err = scanError(&p.s, "Unexpected character "+strconv.QuoteRune(tok))
		}

		if p.err != nil {
			return p.err
		}

	S: // now we only have ints or idents
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
					p.err = scanError(&p.s, "Unexpected label definition as argument: "+s)
					break S
				}
				n := s[1:]
				if len(n) == 0 {
					p.err = scanError(&p.s, "Empty label name")
					break S
				}
				if cst, ok := p.consts[n]; ok {
					p.err = scanError(&p.s, "Label redefinition:"+n+", prefiously defined as a constant here:"+cst.pos.String())
					break S
				}
				if l, ok := p.labels[n]; ok {
					if l.address != -1 {
						p.err = scanError(&p.s, "Label redefinition: "+n+", previous definition here:"+l.pos.String())
					}
					l.address = p.pc
					l.pos = p.s.Pos()
				} else {
					p.labels[n] = &label{
						labelSite{p.s.Pos(), p.pc},
						nil,
					}
				}
			case '.':
				if state != 0 {
					p.err = scanError(&p.s, "Unexpected directive as argument: "+s)
					break S
				}
				switch s {
				case ".org":
					state = 2
				case ".dat":
					state = 1
				case ".equ":
					t := p.s.Scan()
					if t != scanner.Ident {
						p.err = scanError(&p.s, ".equ: expected identifier, got "+p.s.TokenText())
						break S
					}
					p.cstName = p.s.TokenText()
					if l, ok := p.labels[p.cstName]; ok {
						p.err = scanError(&p.s, ".equ: redifinition of "+p.cstName+", previously defined/used as a label: here: "+l.pos.String())
						break S
					}
					p.cstPos = p.s.Pos()
					state = 3
				default: // should use this to define local labels
					p.err = scanError(&p.s, "Unknown dot directive: "+s)
				}
			default:
				if s == "(" {
					// skip comments
					for ; p.err == nil && tok != scanner.EOF && (tok != scanner.Ident || p.s.TokenText() != ")"); tok = p.s.Scan() {
					}
					break S
				}
				if state >= 2 {
					p.err = scanError(&p.s, "Unexpected label as directive argument: "+s)
					break S
				}
				if op, ok := opcodeIndex[s]; ok {
					if state != 0 {
						p.err = scanError(&p.s, "Unexpected opcode as argument: "+s)
						break S
					}
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
					p.useLabel(s)
					p.write(0)
					state = 0
				}
			}
		}
	}

	// write labels
	for n, l := range p.labels {
		if l.address == -1 {
			p.err = fmt.Errorf("Missing label definition for %s, first use here: %s", n, l.uses[0].pos)
			break
		}
		for _, u := range l.uses {
			p.i[u.address] = vm.Cell(l.address)
		}
	}

	return p.err
}
