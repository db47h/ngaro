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

// Package ngi - or ngaro-internal with some commonly used stuff.
package ngi

import (
	"io"

	"github.com/pkg/errors"
)

// ErrWriter is a simple wrapper to track io errors. Write will keep returning
// the last error over and over.
type ErrWriter struct {
	w   io.Writer
	Err error
}

func (w *ErrWriter) Write(p []byte) (n int, err error) {
	if w.Err != nil {
		return 0, w.Err
	}
	n, err = w.w.Write(p)
	if err != nil {
		w.Err = errors.Wrap(err, "write failed")
	}
	return n, w.Err
}

// NewErrWriter returns a new ErrWriter.
func NewErrWriter(w io.Writer) *ErrWriter {
	return &ErrWriter{w, nil}
}
