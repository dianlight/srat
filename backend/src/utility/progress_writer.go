package utility

import (
	"io"
	"sync/atomic"
)

type ProgressWriter struct {
	w    io.Writer
	n    atomic.Int64
	P    chan int
	size int
}

// NewProgressWriter creates a new ProgressWriter that wraps the provided io.Writer.
func NewProgressWriter(w io.Writer, size int) *ProgressWriter {
	return &ProgressWriter{w: w, P: make(chan int), size: size}
}

// Write writes the provided bytes to the underlying writer and updates the progress counter.
func (w *ProgressWriter) Write(b []byte) (n int, err error) {
	n, err = w.Write(b)
	w.P <- n
	w.n.Add(int64(n))
	return n, err
}

// N returns the total number of bytes written by the ProgressWriter.
func (w *ProgressWriter) Written() int64 {
	return w.n.Load()
}

func (w *ProgressWriter) Percent() int8 {
	ret := int8(float64(w.Written()) / float64(w.size) * 100)
	if ret >= 100 {
		ret = 100
	}
	return ret
}
