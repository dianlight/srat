package utility

import (
	"io"
	"sync/atomic"
)

type ProgressWriter struct { // FIXME: Don't work!!!
	w io.Writer
	n atomic.Int64
}

// NewProgressWriter creates a new ProgressWriter that wraps the provided io.Writer.
func NewProgressWriter(w io.Writer) *ProgressWriter {
	return &ProgressWriter{w: w}
}

// Write writes the provided bytes to the underlying writer and updates the progress counter.
func (w *ProgressWriter) Write(b []byte) (n int, err error) {
	n, err = w.Write(b)
	w.n.Add(int64(n))
	return n, err
}

// N returns the total number of bytes written by the ProgressWriter.
func (w *ProgressWriter) N() int64 {
	return w.n.Load()
}
