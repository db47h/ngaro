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
	"io"
	"os"
	"strings"
	"testing"

	"github.com/db47h/ngaro/vm"
)

func Test_io_GetEnv(t *testing.T) {
	var b = bytes.NewBuffer(nil)
	_, err := runImageFile(retroImage,
		vm.Output(vm.NewVT100Terminal(b, nil, nil)),
		vm.Input(strings.NewReader(": pEnv here dup push swap getEnv cr pop puts bye ; \"PATH\" pEnv ")))
	if err != nil {
		panic(err)
	}
	out := bytes.Split(b.Bytes(), []byte{'\n'})
	envRetro := string(out[len(out)-2])
	envGo := os.Getenv("PATH")
	assertEqual(t, "GetEnv", envGo, envRetro)
}

func Test_io_Files(t *testing.T) {
	err := os.Chdir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir("..")
	var b = bytes.NewBuffer(nil)
	_, err = runImageFile("retroImage",
		vm.Output(vm.NewVT100Terminal(b, nil, nil)),
		vm.Input(strings.NewReader("\"files.rx\" :include\n")))
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(b.Bytes(), []byte{'\n'})
	assertEqual(t, "io_Files", "51 tests run: 51 passed, 0 failed.", string(lines[len(lines)-5]))
	assertEqual(t, "io_Files", "15 words checked, 0 words unchecked, 0 i/o words ignored.", string(lines[len(lines)-4]))

	// try to open a file with a dummy mode
	i, err := runAsmImage(`jump start
		:fileName .dat 'r' .dat 'e' .dat 't' .dat 'r' .dat 'o' .dat 'I' .dat 'm' .dat 'a' .dat 'g' .dat 'e' .dat 0
		.org 32
		:io dup push out 0 0 out wait pop in ;
		:start
			lit fileName 0 -1 4 io dup	( open retroImage, should work and return fd = 1 )
			-4 4 io 					( close, should work and return 0 )
			lit fileName 77 -1 4 io		( should fail )`,
		"io_Caps")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	assertEqualI(t, "io_Files data stack size", 3, i.Depth())
	assertEqualI(t, "io_Files dummy mode", 0, int(i.Pop()))
	assertEqualI(t, "io_Files close", 0, int(i.Pop()))
	assertEqualI(t, "io_Files fd", 1, int(i.Pop()))
}

func Test_io_Stacks(t *testing.T) {
	i, err := runAsmImage("-16 5 out 0 0 out wait 5 in -17 5 out 0 0 out wait 5 in", "io_Stacks",
		vm.DataSize(24), vm.AddressSize(42))
	if err != nil {
		t.Fatal(err)
	}
	assertEqualI(t, "io_Stacks", 42, int(i.Pop()))
	assertEqualI(t, "io_Stacks", 24, int(i.Pop()))
}

func Test_io_Caps(t *testing.T) {
	// TODO: although the VM should return a correct value for endianness,
	// the test will fail on BigEndian platforms
	i, err := runAsmImage(`jump start
		.org 32
		:io dup 3 ! out 0 0 out wait 3 @ in ;
		:start
			-6 5 io ( rstack size should be 1 (inside :io) )
			42 push 42 push -6 5 io ( rstack size should be 3 (+1 inside :io) )
			-13 5 io ( cell bits )
			-14 5 io ( endianness )
			-15 5 io ( port 8 enabled )
			 1 1 io ( will cause EOF on nil input )`,
		"io_Caps", vm.Output(vm.NewVT100Terminal(bytes.NewBuffer(nil), nil, nil)))
	if err != io.EOF {
		t.Fatalf("Unexpected error: %v", err)
	}
	assertEqualI(t, "io_Caps port 8", -1, int(i.Pop()))
	assertEqualI(t, "io_Caps endian", 0, int(i.Pop()))
	assertEqualI(t, "io_Caps Cell bits", 32, int(i.Pop()))
	assertEqualI(t, "io_Caps rstack", 3, int(i.Pop()))
	assertEqualI(t, "io_Caps rstack", 1, int(i.Pop()))
}

// Test default In handler (not actually used in core for perf reasons).
func TestVM_In(t *testing.T) {
	i, err := runAsmImage(`20 in 42 20 out 20 in 20 in`,
		"VM_In", vm.BindInHandler(20, (*vm.Instance).In))
	if err != nil {
		t.Fatal(err)
	}
	assertEqualI(t, "VM_In", 0, int(i.Pop()))
	assertEqualI(t, "VM_In", 42, int(i.Pop()))
	assertEqualI(t, "VM_In", 0, int(i.Pop()))
}

func Test_multireader(t *testing.T) {
	i, err := runAsmImage(`jump start
		.org 32
		:io dup push out 0 0 out wait pop in ;
		:start
		1 1 io ( read from input until EOF )
		jump start`,
		"multireader",
		vm.Input(strings.NewReader("56")),
		vm.Input(strings.NewReader("34")),
		vm.Input(strings.NewReader("12")))
	if err != io.EOF {
		t.Fatalf("Unexpected error: %v", err)
	}
	for n := 6; n > 0; n-- {
		assertEqualI(t, "io_multireader", n+48, int(i.Pop()))
	}
}
