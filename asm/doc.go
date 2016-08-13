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

// Package asm provides utility functions to assemble and disassemble Ngaro
// VM code.
//
// Supported assembler mnemonics:
//
//	TOS is the value on top of the data stack. NOS is the next value on the data stack.
//	Instructions with a check mark in the "arg" column expect an argument in the cell
//	following them.
//
//	opcode	asm	arg	stack	description
//	------	---	---	-----	------------------------------------------------------------------------
//	0	nop			no-op
//	1	lit	✓	-n	place next value in next cell on TOS
//	2	dup		n-nn	duplicate TOS
//	3	drop		n-	drop TOS
//	4	swap		xy-yx	swap TOS and NOS
//	5	push		n-	push TOS to address stack
//	6	pop		-n	pop value on top of address stack and place it on TOS
//	7	loop	✓	n-?	decrement TOS. If >0 jump to address in next cell, else drop TOS and do nothing
//	8	jump	✓		jump to address in next cell
//	9	;			return: pop address from address stack, add 1 and jump to it.
//	10	>jump	✓	xy-	jump to address in next cell if NOS > TOS
//	11	<jump	✓	xy-	jump to address in next cell if NOS < TOS
//	12	!jump	✓	xy-	jump to address in next cell if NOS != TOS
//	13	=jump	✓	xy-	jump to address in next cell if NOS == TOS
//	14	@		a-n	fetch: get the value at the address on TOS and place it on TOS.
//	15	!		na-	store: store the value in NOS at address in TOS
//	16	+		xy-z	add NOS to TOS and place result on TOS
//	17	-		xy-z	subtract NOS from TOS and place result on TOS
//	18	*		xy-z	multiply NOS with TOS and place result on TOS
//	19	/mod		xy-rq	divide TOS by NOS and place remainder in NOS, quotient in TOS
//	20	and		xy-z	do a logical and of NOS and TOS and place result on TOS
//	21	or		xy-z	do a logical or of NOS and TOS and place result on TOS
//	22	xor		xy-z	do a logical xor of NOS and TOS and place result on TOS
//	23	<<		xy-z	do a logical left shift of NOS by TOS and place result on TOS
//	24	>>		xy-z	do an arithmetic right shift of NOS by TOS and place result on TOS
//	25	0;		n-?	ZeroExit: if TOS is 0, drop it and do a return, else do nothing
//	26	1+		n-n	increment tos
//	27	1-		n-n	decrement tos
//	28	in		p-n	I/O in (see Ngaro VM spec)
//	29	out		np-	I/O out (see Ngaro VM spec)
//	30	wait		?-	I/O wait (see Ngaro VM spec)
//
// Comments:
//
// Comments are placed between parentheses, i.e. '(' and ')'. The body of the
// comment must be separated from the enclosing parentheses by a space. That is:
//
// Some valid comments:
//
//	( this is a valid comment )
//	( this is a
//	  rather long
//	  multiline comment )
//
// The following ae invalid comments:
//
//	(this will be seen by the parser as label "(this" and will not work )
//	( comments may ( not be nested ) here, the parser will complain trying to resolve
//	  "here," as a label )
//
// Literals and label/const identifiers:
//
// The parser behaves almost like a Forth parser: input is split at white space
// (space, tab or new line) into tokens. The parser then does the following:
//
//	- If a token can be converted to a Go integer (see strconv.ParseInt), it will
//	  be converted to an integer literal.
//	- If it is a Go character literal between single quotes, it will be converted to
//	  the corresponding integer literal. Watch out with unicode chars: they will be
//	  convberted to the proper rune (int32), but they are not natively supported by
//	  the VM I/O code.
//	- If a token is the name of a defined constant, it will be replaced internally by
//	  the constant's value and can be used anywhere an integer literal is expected.
//
//	- Then name resolution applies:
//	  - if an instruction is expected, the token is looked up in the assembler
//	    mnemonics and if no match is found, it is considered to be a label.
//	  - if an argument is expected, the token is always considered a label.
//
// You may therefore define unusual labels or constant names (at least for Go
// programmers) such as "2dup", "(weird" or "end-weird)". Also, more than one
// instruction may appear on the same line and comments can be placed anywhere
// between instructions.
//
// Implicit "lit":
//
// Where the parser is expecting an instruction, integer literals, character
// literals and constants will be compiled with an implicit "lit":
//
//	lit 42
//	42	( will compile as "lit 42", just like above )
//	( like ) 'a' ( compiles as ) lit 'a' ( which in fact compiles as ) lit 97
//
// Labels:
//
// Labels are defined by prefixing them with a colon (:) and can be used as address
// in any lit, jump or loop instruction (without the ':' prefix). For example:
//
//	foo		( forward references are ok. This will be compiled as a call to foo )
//	lit foo		( this will compile as lit <address of foo>. This is actually the
//			  only way to place the address of a label on the stack. )
//
//	:foo		( foo defined here )
//	nop
//	;
//
//	:bar	nop	( label definitions can be grouped with other instructions on the same line )
//		;
//
//	:foobar	nop ;	( we can actually place any number of instructions on the same line )
//
// Local labels:
//
// Local labels work in the same way as in the GNU assembler. They are defined
// as a colon followed by a sequence of digits (i.e. :007, :0, :42). Although
// they can be defined multiple times, the compiler internally assigns them a
// unique name of the form N·counter (the middle character is '\u00b7').
// References to such labels must be suffixed with either a '-' (meaning backward
// reference to the last definition of this label), or a '+' (meaning a forward
// reference to the next definition of this label). For example, in the following
// code:
//
//	:1	jump 1+	( not to be confused with the '1+' mnemonic. Here it means next occurrence of :1 )
//	:2	jump 1-
//	:1	jump 2+
//	:2	jump 1-
//
// the labels will be internally converted to:
//
//	:1·1	jump 1·2
//	:2·1	jump 1·1
//	:1·2	jump 2·1
//	:2·2	jump 1·2
//
// As a consequence, you should not use or define labels of the form N·N where N is
// any non-empty sequence of difigts. This also prevents the definition of labels
// of the form N+ or N- because they will not be addressable.
//
// Please note that the parser does not prevent you either from using/defining labels
// with the same name as instructions. The only caveat, besides confusing yourself,
// is that you will not be able to use implicit calls to such labels:
//
//	:drop	'D' 1 1 out 0 0 out wait ( print 'D' )
//		drop ;	( this will not loop forever, drop will be compiled as opcode 3, not a call )
//	drop		( still opcode 3 )
//	.dat drop	( will compile an implicit call to our custom drop )
//
// Assembler directives:
//
// The assembler supports the following directives:
//
//	.equ <IDENTIFIER> <value>
//
// defines a constant value. <IDENTIFIER> can be any valid identifier (any
// combination of letters, symbols, digits and punctuation). The value must be
// an integer value, named constant or character literal.
//
//	.org <value>
//
// Will place the next instruction at the address specified by the given integer
// literal or named constant.
//
//	.dat <value>
//
// Will compile the specified integer value, named constant or character literal
// as-is (i.e. with no implicit "lit"). This is primarily used used for data
// storage structures:
//
//	:table	.dat 65
//		.dat 'B'
//
// The cells at addresses table+0 and table+1 will contain 65 and 66 respectively.
package asm
