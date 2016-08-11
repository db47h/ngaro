package main

import (
	"io"
	"strconv"

	"github.com/db47h/ngaro/vm"
)

type errWriter struct {
	w   io.Writer
	err error
}

func (w *errWriter) Write(p []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err = w.w.Write(p)
	if err != nil {
		w.err = err
	}
	return n, err
}

func (w *errWriter) dumpSlice(a []vm.Cell) error {
	l := len(a) - 1
	if l >= 0 {
		for i := 0; i < l; i++ {
			io.WriteString(w, strconv.Itoa(int(a[i])))
			w.Write([]byte{' '})
		}
		io.WriteString(w, strconv.Itoa(int(a[l])))
	}
	return w.err
}

// Dump dumps the virtual machine stacks and image to the specified io.Writer.
func dumpVM(i *vm.Instance, size int, w io.Writer) error {
	ew := &errWriter{w: w}
	ew.Write([]byte{'\x1C'})
	ew.dumpSlice(i.Data())
	ew.Write([]byte{'\x1D'})
	ew.dumpSlice(i.Address())
	ew.Write([]byte{'\x1D'})
	return ew.dumpSlice(i.Image[:size])
}
