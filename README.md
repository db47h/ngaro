[![Build Status](https://travis-ci.org/db47h/ngaro.svg?branch=master)](https://travis-ci.org/db47h/ngaro)
[![Go Report Card](https://goreportcard.com/badge/github.com/db47h/ngaro)](https://goreportcard.com/report/github.com/db47h/ngaro)  [![GoDoc](https://godoc.org/github.com/db47h/ngaro/vm?status.svg)](https://godoc.org/github.com/db47h/ngaro/vm)

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
communication between Retro programs and Go programs via custom I/O handlers
(i.e. scripting Go programs in Retro) as well as supporting custom opcodes at
the VM level. The package examples demonstrate various use cases. For more
details on I/O handling in the Ngaro VM, please refer to
http://retroforth.org/docs/The_Ngaro_Virtual_Machine.html.

This implementation passes all tests from the retro-language test suite and
its performance when running tests/core.rx is slightly better than with the
reference implementations:

	1.20s for this implementation, compiled with Go 1.7rc6.
	1.30s for the reference Go implementation, compiled with Go 1.7rc6
	2.22s for the reference C implementation, compiled with gcc-5.4 -O3 -fomit-frame-pointer

For all intents and purposes, the VM behaves according to the specification.
With one exception: if you implement custom opcodes, be aware that for
performance reasons, the PC (aka. Instruction Pointer) is not incremented in
a single place; rather each opcode deals with the PC as needed. Users of
custom opcodes will need to take care of updating the PC accordingly. This
should be of no concern to other users, even with custom I/O  handlers.
Should you find that the VM does not behave according to the spec, please
file a bug report.
	
