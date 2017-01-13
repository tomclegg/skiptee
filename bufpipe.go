package skiptee

import (
	"io"
)

type bufpipe struct {
	buffer    chan []byte
	newBuffer chan chan []byte
	stopped   chan struct{}
	err       error
}

func (p *bufpipe) write(data []byte) error {
	if len(p.newBuffer) > 0 {
		// WriteTo has not even noticed the current buffer
		// channel yet.
		return nil
	}
	select {
	case <-p.stopped:
		return io.EOF
	case p.buffer <- data:
		return nil
	default:
		p.buffer = make(chan []byte, len(p.buffer))
		p.newBuffer <- p.buffer
		return nil
	}
}

func (p *bufpipe) close() {
	close(p.buffer)
}

func (p *bufpipe) writeTo(w io.Writer, ready chan struct{}) (n int64, err error) {
	buffer := <-p.newBuffer
	if ready != nil {
		close(ready)
	}
	defer close(p.stopped)
	for {
		select {
		case buffer = <-p.newBuffer:
			continue
		default:
		}
		select {
		case buffer = <-p.newBuffer:
			continue
		case data := <-buffer:
			if data == nil {
				return
			}
			var bytes int
			bytes, err = w.Write(data)
			n += int64(bytes)
			if err != nil {
				return
			}
		}
	}
}
