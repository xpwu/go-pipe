package pipe

// 系统的io/pipe.go 没有超时设计，以及获取错误信息时有一个小bug，所以在系统的代码基础上，修改能满足实际使用的io/pipe版本

import (
	"io"
	"os"
	"sync"
	"time"
)

// 为了方便跳转到系统的pipe文件
func _() {
	io.Pipe()
}

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Pipe adapter to connect code expecting an io.Reader
// with code expecting an io.Writer.

// onceError is an object that will only store an error once.
type onceError struct {
	sync.Mutex // guards following
	err        error
}

func (a *onceError) store(err error) {
	a.Lock()
	defer a.Unlock()
	if a.err != nil {
		return
	}
	a.err = err
}
func (a *onceError) load() error {
	a.Lock()
	defer a.Unlock()
	return a.err
}

// ErrClosedPipe is the error used for read or write operations on a closed pipe.
//var ErrClosedPipe = errors.New("io: read/write on closed pipe")

// A pipe is the shared pipe structure underlying PipeReader and PipeWriter.
type ioPipe struct {
	wrMu sync.Mutex // Serializes Write operations
	wrCh chan []byte
	rdCh chan int

	once sync.Once // Protects closing done
	done chan struct{}
	rerr onceError
	werr onceError

	rTimeout time.Time
	wTimeout time.Time

	r func(b []byte) (n int, err error)
	w func(b []byte) (n int, err error)
}

func newPipe() *ioPipe {
	p := &ioPipe{
		wrCh: make(chan []byte),
		rdCh: make(chan int),
		done: make(chan struct{}),
	}
	p.r = p.readNoTimeout
	p.w = p.writeNoTimeout

	return p
}

func (p *ioPipe) setReadDeadline(t time.Time) {
	p.rTimeout = t
	if t.IsZero() {
		p.r = p.readNoTimeout
		return
	}
	p.r = p.readWithTimeout
}

func (p *ioPipe) read(b []byte) (n int, err error) {
	return p.r(b)
}

func (p *ioPipe) readNoTimeout(b []byte) (n int, err error) {
	select {
	case <-p.done:
		return 0, p.readCloseError()
	default:
	}

	select {
	case bw := <-p.wrCh:
		nr := copy(b, bw)
		p.rdCh <- nr
		return nr, nil
	case <-p.done:
		return 0, p.readCloseError()
	}
}

func (p *ioPipe) readWithTimeout(b []byte) (n int, err error) {
	d := time.Until(p.rTimeout)
	if d <= 0 {
		return 0, os.ErrDeadlineExceeded
	}

	select {
	case <-p.done:
		return 0, p.readCloseError()
	default:
	}

	timer := time.NewTimer(d)

	select {
	case bw := <-p.wrCh:
    timer.Stop()
		nr := copy(b, bw)
		p.rdCh <- nr
		return nr, nil
	case <-p.done:
		timer.Stop()
		return 0, p.readCloseError()
	case <-timer.C:
		return 0, os.ErrDeadlineExceeded
	}
}

func (p *ioPipe) readCloseError() error {
	if rerr := p.rerr.load(); rerr != nil {
		return rerr
	}

	return io.ErrClosedPipe
}

func (p *ioPipe) closeRead(err error) error {
	if err == nil {
		err = io.ErrClosedPipe
	}
	p.rerr.store(err)
	p.once.Do(func() { close(p.done) })
	return nil
}

func (p *ioPipe) setWriteDeadline(t time.Time) {
	p.wTimeout = t
	if t.IsZero() {
		p.w = p.writeNoTimeout
		return
	}
	p.w = p.writeWithTimeout
}

func (p *ioPipe) write(b []byte) (n int, err error) {
	return p.w(b)
}

func (p *ioPipe) writeNoTimeout(b []byte) (n int, err error) {
	select {
	case <-p.done:
		return 0, p.writeCloseError()
	default:
		p.wrMu.Lock()
		defer p.wrMu.Unlock()
	}

	for once := true; once || len(b) > 0; once = false {
		select {
		case p.wrCh <- b:
			nw := <-p.rdCh
			b = b[nw:]
			n += nw
		case <-p.done:
			return n, p.writeCloseError()
		}
	}
	return n, nil
}

func (p *ioPipe) writeWithTimeout(b []byte) (n int, err error) {
	d := time.Until(p.wTimeout)
	if d <= 0 {
		return 0, os.ErrDeadlineExceeded
	}

	select {
	case <-p.done:
		return 0, p.writeCloseError()
	default:
		p.wrMu.Lock()
		defer p.wrMu.Unlock()
	}

	timer := time.NewTimer(d)

	for once := true; once || len(b) > 0; once = false {
		select {
		case p.wrCh <- b:
			nw := <-p.rdCh
			b = b[nw:]
			n += nw
		case <-p.done:
			timer.Stop()
			return n, p.writeCloseError()
		case <-timer.C:
			return n, os.ErrDeadlineExceeded
		}
	}

  timer.Stop()
	return n, nil
}

func (p *ioPipe) writeCloseError() error {
	if werr := p.werr.load(); werr != nil {
		return werr
	}

	return io.ErrClosedPipe
}

func (p *ioPipe) closeWrite(err error) error {
	if err == nil {
		err = io.EOF
	}
	p.werr.store(err)
	p.once.Do(func() { close(p.done) })
	return nil
}

// A PipeReader is the read half of a pipe.
type Reader struct {
	p *ioPipe
}

// Read implements the standard Read interface:
// it reads data from the pipe, blocking until a writer
// arrives or the write end is closed.
// If the write end is closed with an error, that error is
// returned as err; otherwise err is EOF.
func (r *Reader) Read(data []byte) (n int, err error) {
	return r.p.read(data)
}

// Close closes the reader; subsequent writes to the
// write half of the pipe will return the error ErrClosedPipe.
func (r *Reader) Close() error {
	return r.CloseWithError(nil)
}

// CloseWithError closes the reader; subsequent writes
// to the write half of the pipe will return the error err.
//
// CloseWithError never overwrites the previous error if it exists
// and always returns nil.
func (r *Reader) CloseWithError(err error) error {
	return r.p.closeRead(err)
}

func (r *Reader) SetDeadline(t time.Time) {
	r.p.setReadDeadline(t)
}

// A PipeWriter is the write half of a pipe.
type Writer struct {
	p *ioPipe
}

// Write implements the standard Write interface:
// it writes data to the pipe, blocking until one or more readers
// have consumed all the data or the read end is closed.
// If the read end is closed with an error, that err is
// returned as err; otherwise err is ErrClosedPipe.
func (w *Writer) Write(data []byte) (n int, err error) {
	return w.p.write(data)
}

// Close closes the writer; subsequent reads from the
// read half of the pipe will return no bytes and EOF.
func (w *Writer) Close() error {
	return w.CloseWithError(nil)
}

// CloseWithError closes the writer; subsequent reads from the
// read half of the pipe will return no bytes and the error err,
// or EOF if err is nil.
//
// CloseWithError never overwrites the previous error if it exists
// and always returns nil.
func (w *Writer) CloseWithError(err error) error {
	return w.p.closeWrite(err)
}

func (w *Writer) SetDeadline(t time.Time) {
	w.p.setWriteDeadline(t)
}

// Pipe creates a synchronous in-memory pipe.
// It can be used to connect code expecting an io.Reader
// with code expecting an io.Writer.
//
// Reads and Writes on the pipe are matched one to one
// except when multiple Reads are needed to consume a single Write.
// That is, each Write to the PipeWriter blocks until it has satisfied
// one or more Reads from the PipeReader that fully consume
// the written data.
// The data is copied directly from the Write to the corresponding
// Read (or Reads); there is no internal buffering.
//
// It is safe to call Read and Write in parallel with each other or with Close.
// Parallel calls to Read and parallel calls to Write are also safe:
// the individual calls will be gated sequentially.
func Pipe() (*Reader, *Writer) {
	p := newPipe()
	return &Reader{p}, &Writer{p}
}
