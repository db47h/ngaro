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
	"strings"
	"testing"

	"github.com/db47h/ngaro/asm"
)

// check some errors. We're not checking the messages, rather that they point at
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
	[un]		( undefined )
	jump :lab	( valid but undef'ed )
	jump 0001-
	.org 001+		( stupid )
	.org :foo
	'yo'
	'\x'
	>jump .zoo
::foo
:001
	`
	_, err = asm.Assemble("test_errors", strings.NewReader(code))

	errs := err.(asm.ErrAsm)
	// locate and match errors in source code
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

	// TODO: test uncovered errors (about 5 are easy)
}
