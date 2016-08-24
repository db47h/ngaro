[![Build Status](https://travis-ci.org/db47h/ngaro.svg?branch=master)](https://travis-ci.org/db47h/ngaro)
[![Go Report Card](https://goreportcard.com/badge/github.com/db47h/ngaro)](https://goreportcard.com/report/github.com/db47h/ngaro)
[![Coverage Status](https://coveralls.io/repos/github/db47h/ngaro/badge.svg)](https://coveralls.io/github/db47h/ngaro)
[![GoDoc](https://godoc.org/github.com/db47h/ngaro/vm?status.svg)](https://godoc.org/github.com/db47h/ngaro/vm)

# Ngaro Go

## <a name="pkg-overview">Overview</a>
This is an embeddable Go implementation of the [Ngaro Virtual Machine](http://retroforth.org/docs/The_Ngaro_Virtual_Machine.html).

This repository contains the embeddable [virtual
machine](https://godoc.org/github.com/db47h/ngaro/vm), a rudimentary
[symbolic assembler](https://godoc.org/github.com/db47h/ngaro/asm)
for easy bootstrapping of projects written in Ngaro machine language, and the
[retro](https://godoc.org/github.com/db47h/ngaro/cmd/retro) command
line tool that can be used as a replacement for the Retro reference
implementations.

Please visit http://forthworks.com/retro/ to get you started about the Retro
language and the Ngaro Virtual Machine.

The main purpose of this implementation is to allow customization and
communication between Retro programs and Go programs via custom opcodes and
I/O handlers (i.e. scripting Go programs in Retro). The package examples
demonstrate various use cases. For more details on I/O handling in the Ngaro
VM, please refer to http://retroforth.org/docs/The_Ngaro_Virtual_Machine.html.

Another goal is to make the VM core as neutral as possible regarding the higher
level language running on it. Some Retro specific behaviors have been moved from
the VM to the retro command line tool, like shrinking memory image dumps that
relies on reading the memory image size at a specific location in memory.

Custom opcodes are implemented by intercepting implicit calls to negative memory
addresses. This allows the VM to be fully backwards compatible with existing
Retro images while still providing enhanced capabilities. The maximum number of
addressable cells is 2^31 when running in 32 bits mode (that's 8GiB or memory on
the host). The range [-2^31 - 1, -1] is available for custom opcodes.

This implementation passes all tests from the retro-language test suite and
its performance when running tests/core.rx is slightly better than with the
reference implementations:

	1.08s for this implementation, no custom opcodes, compiled with Go 1.7, linux/amd64
	1.15s for the reference assembly implementation, linux/386
	1.30s for the reference Go implementation, compiled with Go 1.7, linux/amd64
	2.00s for the reference C implementation, compiled with gcc-5.4 -O3 -fomit-frame-pointer

Yes, Go 1.7's new SSA backend is THAT good on this type of workload :)

For all intents and purposes, the VM behaves according to the specification.
This is of particular importance to implementors of custom opcodes: the VM
always increments the PC after each opcode, thus opcodes altering the PC must
adjust it accordingly (i.e. set it to its real target minus one).

## Installing

Install the retro command line tool:

	go get -u github.com/db47h/ngaro/cmd/retro

Test:

	go test -i github.com/db47h/ngaro/vm
	go test -v github.com/db47h/ngaro/vm/...

Build a retroImage:

	cd $GOPATH/github.com/db47h/ngaro/cmd/retro
	make retroImage

Test the retro command line tool:

	./retro --with vm/testdata/core.rx

Should generate a lot of output. Just check that the last lines look like this:

	ok   summary
	360 tests run: 360 passed, 0 failed.
	186 words checked, 0 words unchecked, 37 i/o words ignored.

	ok  bye

## Support for 32/64 bits memory images on all architectures

Since v2.0.0, the default Cell type (the base data type in Ngaro VM) is Go's
int. This means that depending on the target you compile for, it will be either
32 or 64 bits. The retro command line tool supports loading and saving retro
memory images where Cells can be either size. For example, to quickly get
started you can do this:

	echo "save bye" | \
	retro -image vm/testdata/retroImage -ibits 32 -o retroImage

This will load the memory image file `vm/testdata/retroImage` which we know to
be encoded using 32 bits cells, and save it in the current directory with
whatever encoding is the default for your platform. You could also force a
specific output Cell size with the `-obits` flag.

Loading and saving with encodings different from the target platform is safe:
it will work or generate an error, but never create a corrupted memory
image file. For example, with a 64 bits retro binary, saving to 32 bits cells
will check that written values fit in a 32 bit int. If not, it will generate an
error.

If for some reason you need a specific cell size, regardless of the target
platform's native int size, you can force it by compiling with the tags
`ngaro32` or `ngaro64`:

	go install -tags ngaro32 github.com/db47h/ngaro/cmd/retro

will build a version of retro that uses 32 bits cells, regardless of
`GOOS`/`GOARCH`. Likewise, the `ngaro64` tag will force 64 bits cells, even on
32 bits targets (it'll be twice as slow though).

## Releases

This project uses [semantic
versioning](http://dave.cheney.net/2016/06/24/gophers-please-tag-your-releases)
and tries to adhere to it.

See the [releases page](https://github.com/db47h/ngaro/releases).

For a detailed change log, see the [commit log](https://github.com/db47h/ngaro/commits/master).

## License

This project is Copyright 2016 Denis Bernard <db047h@gmail.com>, licensed under
the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0).

The Retro language and Ngaro Virtual Machine are Copyright (c) 2008-2016 Charles
Childers (and many others), licensed under the ISC license. See the file
LICENSE-RETRO at the root of this repository for more details as well as a full
list of contributors.

Note that all files in the \_misc and vm/testdata folders are verbatim copies
from the retro-language project. As such, only Retro's ISC license applies to
these files.
