package skiptee

import (
	"io"
	"sync"
)

type Tee interface {
	io.WriteCloser
	io.WriterTo
	NewReader() io.ReadCloser
}

type tee struct {
	bufsize int
	outputs map[*bufpipe]struct{}
	lock    sync.Mutex
}

func New(bufsize int) Tee {
	return &tee{
		bufsize: bufsize,
		outputs: make(map[*bufpipe]struct{}),
	}
}

func (t *tee) NewReader() io.ReadCloser {
	r, w := io.Pipe()
	ready := make(chan struct{})
	go func() {
		t.add().writeTo(w, ready)
		w.Close()
	}()
	<-ready
	return r
}

func (t *tee) Write(buf []byte) (int, error) {
	bufcopy := make([]byte, len(buf))
	copy(bufcopy, buf)
	t.lock.Lock()
	defer t.lock.Unlock()
	for p := range t.outputs {
		if err := p.write(bufcopy); err != nil {
			p.close()
			delete(t.outputs, p)
		}
	}
	return len(buf), nil
}

func (t *tee) Close() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	for p := range t.outputs {
		p.close()
	}
	t.outputs = nil
	return nil
}

func (t *tee) add() *bufpipe {
	p := &bufpipe{
		buffer:    make(chan []byte, t.bufsize),
		newBuffer: make(chan chan []byte, 1),
		stopped:   make(chan struct{}),
	}
	p.newBuffer <- p.buffer
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.outputs == nil {
		p.close()
	} else {
		t.outputs[p] = struct{}{}
	}
	return p
}

func (t *tee) remove(p *bufpipe) {
	t.lock.Lock()
	defer t.lock.Unlock()
	delete(t.outputs, p)
}

func (t *tee) WriteTo(w io.Writer) (int64, error) {
	p := t.add()
	defer t.remove(p)
	return p.writeTo(w, nil)
}
