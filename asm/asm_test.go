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

package asm_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/db47h/ngaro/asm"
)

// check some errors. We're not checking the whole messages, rather that they point at
// the correct place.
func TestAssemble_errors(t *testing.T) {
	// scanner error
	code := "\x80AA"
	_, err := asm.Assemble("test_errors", strings.NewReader(code))
	// BUG: this illegal rune should actually be skipped and not returned as part of a token.
	expected := "test_errors:1:1: illegal UTF-8 encoding\ntest_errors:1:1: Undefined label \x80AA"
	got := err.Error()
	if expected != got {
		t.Errorf("Expected: %s\nGot: %s\n", expected, got)
	}

	// others
	code = `
	jump :lab	( valid but undef'ed )
	jump 0001-
	.org 001+		( stupid )
	.org :foo
	'yo'
	'\x'
	.equ ABC  567
	.equ 1234 567
	>jump .zoo
	.dat :lab ( only way to call a colon prefixed label )
::foo
:001
	`
	_, err = asm.Assemble("test_errors", strings.NewReader(code))

	errs := err.(asm.ErrAsm)
	// locate and match errors in source code
	if len(errs) < 10 {
		t.Errorf("Expected 10 errors, got %d", len(errs))
	}
	for _, e := range errs {
		o := e.Pos.Offset
		end := o + 4
		if end > len(code) {
			end = len(code)
		}

		if !strings.HasSuffix(e.Msg, code[o:end]) {
			t.Errorf("Error message \"%s\" points to %s", e.Msg, code[o:end])
		}
	}
}

func TestAssemble_errors2(t *testing.T) {
	data := []struct {
		name string
		code string
		err  string
	}{
		{"empty_lbl", ":", "empty_lbl:1:1: Empty label name in label definition"},
		{"lbl_cst", ".equ FOO 0 :FOO", "lbl_cst:1:12: Label FOO already defined as a constant\nlbl_cst:1:6: Previous definition of FOO"},
		{"lbl_redef", ":FOO :FOO", "lbl_redef:1:6: Label FOO already defined\nlbl_redef:1:1: Previous definition of FOO"},
		{"cst_label", "bar .equ bar 0 :bar .equ bar 0",
			"cst_label:1:10: Constant bar already used as a label\n" +
				"cst_label:1:1: Previous use of bar\n" +
				"cst_label:1:26: Constant bar already used as a label\n" +
				"cst_label:1:16: Previous use of bar"},
		{"unterm_str", ".dat \"hello\nfoo", "unterm_str:1:12: Unterminated string \"hello\nunterm_str:2:1: Undefined label foo"},
		{"bad_string", ".dat \"hello \\k world\"", "bad_string:1:16: invalid syntax in string \"hello \\k world\""},
	}

	for _, i := range data {
		_, err := asm.Assemble(i.name, strings.NewReader(i.code))
		if err == nil {
			t.Errorf("Test %s: unexpected nil error", i.name)
			continue
		}
		if err.Error() != i.err {
			t.Errorf("Test %s:\nExpected: %v\n     Got: %v", i.name, i.err, err)
		}
	}
}

func Test_strings(t *testing.T) {
	_, err := asm.Assemble("testStrings", strings.NewReader(`
		"1234"
		lit "1234"
		.org "1234"
		.opcode "12345"
		.equ TOTO "123456"
		`))
	if err == nil {
		t.Fatal("Unexpected nil error")
	}
	exp := `testStrings:2:3: string can only be used after a .dat directive
testStrings:3:7: string can only be used after a .dat directive
testStrings:4:8: string can only be used after a .dat directive
testStrings:5:11: Invalid constant or opcode identifier: "12345"
testStrings:6:13: string can only be used after a .dat directive`
	if s := err.Error(); s != exp {
		t.Fatalf("\nExpected:\n%s\nGot:\n%s", exp, s)
	}
	img, err := asm.Assemble("testStrings", strings.NewReader(`
		.dat "hello, world "
		`))
	if err != nil {
		t.Fatal(err)
	}
	s := fmt.Sprintf("%v", img)
	exp = "[104 101 108 108 111 44 32 119 111 114 108 100 32 0]"
	if exp != s {
		t.Fatalf("\nExpected:\n%s\nGot:\n%s", exp, s)
	}
}
