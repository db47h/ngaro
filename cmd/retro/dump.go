package main

import (
	"io"
	"strconv"

	"github.com/db47h/ngaro/internal/ngi"
	"github.com/db47h/ngaro/vm"
)

func dumpSlice(w *ngi.ErrWriter, a []vm.Cell) error {
	l := len(a) - 1
	if l >= 0 {
		for i := 0; i < l; i++ {
			io.WriteString(w, strconv.Itoa(int(a[i])))
			w.Write([]byte{' '})
		}
		io.WriteString(w, strconv.Itoa(int(a[l])))
	}
	return w.Err
}

// Dump dumps the virtual machine stacks and image to the specified io.Writer.
func dumpVM(i *vm.Instance, size int, w io.Writer) error {
	ew := ngi.NewErrWriter(w)
	ew.Write([]byte{'\x1C'})
	dumpSlice(ew, i.Data())
	ew.Write([]byte{'\x1D'})
	dumpSlice(ew, i.Address())
	ew.Write([]byte{'\x1D'})
	return dumpSlice(ew, i.Image[:size])
}
