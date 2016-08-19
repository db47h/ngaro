package asm_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/db47h/ngaro/asm"
)

// Shows off some of the assembler features (although the example assembly program
// is complete non-sense).
func ExampleAssemble() {
	code := `
		( this is a comment. brackets must be separated by spaces )
		
		( a constant definition. Does not generate any code on its own )
		.equ SOMECONST 42
		
		nop
		123			( implicit literal )
		SOMECONST   ( const literal )
		drop
		drop
		foo			( implicit call )
		pop
		lit table	( address of table )
		'x'			( char literal, compiles as lit 'x' )
		
		.org 32 ( set compilation address )
		
:foo	42 bar drop ;
:bar	1+ ;  ( several instructions on the same line )

		.opcode sqrt -1	( test custom opcode )
		sqrt			( should compile like .dat -1 )
		
:table	( data structure )
		.dat -100		( will appear in the disassembly as "call -100" )
		.dat 0666		( octal )
		.dat 0x27		( hex )
		.dat '\u2033'	( unicode char )
		.dat SOMECONST
		.dat foo		( address of some label )
`

	img, err := asm.Assemble("raw_string", strings.NewReader(code))
	if err != nil {
		fmt.Println(err)
		return
	}

	asm.DisassembleAll(img, 0, os.Stdout)

	// Output:
	//          0	nop
	//          1	123
	//          3	42
	//          5	drop
	//          6	drop
	//          7	call 32
	//          8	pop
	//          9	40
	//         11	120
	//         13	nop
	//         14	nop
	//         15	nop
	//         16	nop
	//         17	nop
	//         18	nop
	//         19	nop
	//         20	nop
	//         21	nop
	//         22	nop
	//         23	nop
	//         24	nop
	//         25	nop
	//         26	nop
	//         27	nop
	//         28	nop
	//         29	nop
	//         30	nop
	//         31	nop
	//         32	42
	//         34	call 37
	//         35	drop
	//         36	;
	//         37	1+
	//         38	;
	//         39	call -1
	//         40	call -100
	//         41	call 438
	//         42	call 39
	//         43	call 8243
	//         44	call 42
	//         45	call 32
}

// Disassemble is pretty straightforward. Here we Disassemble a hand crafted
// fibonacci function.
func ExampleDisassemble() {
	fibS := `
	:fib
		push 0 1 pop	( like [ 0 1 ] dip )
		jump 1+		( jump forward to the next :1 )
	:0  push		( local label )
		dup	push
		+
		pop	swap
		pop
	:1  loop 0-		( local label back )
		swap drop ;
		lit		( lit deliberately unterminated at end of image for testing purposes )
		`
	img, err := asm.Assemble("fib", strings.NewReader(fibS))
	if err != nil {
		panic(err)
	}

	for pc := 0; pc < len(img); {
		fmt.Printf("% 4d\t", pc)
		pc, err = asm.Disassemble(img, pc, os.Stdout)
		if err != nil {
			panic(err)
		}
		fmt.Println()
	}

	fmt.Println("Partial disassembly with DisassembleAll:")

	// Partial diasssembly. Set base accordingly so that the address column is correct.
	asm.DisassembleAll(img[15:20], 15, os.Stdout)

	// Output:
	//    0	push
	//    1	0
	//    3	1
	//    5	pop
	//    6	jump 15
	//    8	push
	//    9	dup
	//   10	push
	//   11	+
	//   12	pop
	//   13	swap
	//   14	pop
	//   15	loop 8
	//   17	swap
	//   18	drop
	//   19	;
	//   20	???
	// Partial disassembly with DisassembleAll:
	//         15	loop 8
	//         17	swap
	//         18	drop
	//         19	;
}

// Demonstrates use of local labels
func Example_locals() {
	code := `
	:1	jump 1+
	:2	jump 1-
	:1	jump 2+
	:2	jump 1-
	`

	img, err := asm.Assemble("locals", strings.NewReader(code))
	if err != nil {
		fmt.Println(err)
		return
	}

	asm.DisassembleAll(img, 0, os.Stdout)

	// Output:
	//          0	jump 4
	//          2	jump 0
	//          4	jump 6
	//          6	jump 4
}
