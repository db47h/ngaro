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

//+build !windows

package main

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/pkg/term/termios"
)

// switch terminal to raw IO. we do not use the higher level functions
// of the term package because it doesn't allow the use of existing file
// descriptors, nor does it allow custom termios settings.
func setRawIO() (func(), error) {
	var tios syscall.Termios
	err := termios.Tcgetattr(0, &tios)
	if err != nil {
		return nil, err
	}
	a := tios
	a.Iflag &^= syscall.BRKINT | syscall.ISTRIP | syscall.IXON | syscall.IXOFF
	a.Iflag |= syscall.IGNBRK | syscall.IGNPAR
	a.Lflag &^= syscall.ICANON | syscall.ISIG | syscall.IEXTEN | syscall.ECHO
	a.Cc[syscall.VMIN] = 1
	a.Cc[syscall.VTIME] = 0
	err = termios.Tcsetattr(0, termios.TCSANOW, &a)
	if err != nil {
		// well, try to restore as it was if it errors
		termios.Tcsetattr(0, termios.TCSANOW, &tios)
		return nil, err
	}
	return func() {
		termios.Tcsetattr(0, termios.TCSANOW, &tios)
	}, nil
}

type winsize struct {
	row, col, xpixel, ypixel uint16
}

func ioctl(fd uintptr, request, argp uintptr) (err error) {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, argp)
	if errno != 0 {
		err = errno
	}
	return err
}

func consoleSize(f *os.File) func() (int, int) {
	return func() (int, int) {
		var w winsize
		err := ioctl(f.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&w)))
		if err != nil {
			return 0, 0
		}
		return int(w.col), int(w.row)
	}
}
