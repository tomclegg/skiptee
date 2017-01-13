package skiptee

import (
	"bytes"
	"gopkg.in/check.v1"
	"io"
	"io/ioutil"
	"sync"
	"testing"
	"time"
)

func Test(t *testing.T) { check.TestingT(t) }

type Suite struct{}

var _ = check.Suite(&Suite{})

func (s *Suite) TestReader(c *check.C) {
	t := New(4)
	n, err := t.Write([]byte{1, 2, 3})
	c.Check(n, check.Equals, 3)
	c.Check(err, check.IsNil)

	r := t.NewReader()

	n, err = t.Write([]byte{4, 5, 6, 7})
	c.Check(n, check.Equals, 4)
	c.Check(err, check.IsNil)

	t.Close()

	buf, err := ioutil.ReadAll(r)
	c.Check(buf, check.DeepEquals, []byte{4, 5, 6, 7})
	c.Check(err, check.IsNil)

	n, err = r.Read(buf)
	c.Check(n, check.Equals, 0)
	c.Check(err, check.Equals, io.EOF)

	c.Check(r.Close(), check.IsNil)
}

func (s *Suite) TestOverflow(c *check.C) {
	t := New(4)

	r := make([]io.ReadCloser, 123)
	for i := 0; i < len(r); i++ {
		r[i] = t.NewReader()
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	buf0 := &bytes.Buffer{}
	buf1 := &bytes.Buffer{}
	for i := 0; i < 123; i++ {
		n, err := t.Write([]byte{byte(i)})
		time.Sleep(time.Millisecond)
		c.Check(n, check.Equals, 1)
		c.Check(err, check.IsNil)
		if i == 0 {
			go func() {
				io.Copy(buf0, r[0])
				wg.Done()
			}()
		}
		if i == 100 {
			// r[1] is not ready until the 101st write
			go func() {
				io.Copy(buf1, r[1])
				wg.Done()
			}()
		}
	}
	t.Close()
	wg.Wait()

	checkSequence := func(data []byte) {
		for i, b := range data {
			switch {
			case i == 0:
			case b < data[i-1]+1:
				c.Fatalf("non-increasing sequence at %d: %d,%d", i-1, data[i-1], b)
			case b == data[i-1]+1:
			case b <= data[i-1]+5:
				c.Fatalf("skipped less than a full buffer at %d: %d,%d", i-1, data[i-1], b)
			}
		}
	}
	checkSequence(buf0.Bytes())
	checkSequence(buf1.Bytes())
	c.Assert(buf1.Len() >= 2, check.Equals, true)
	c.Check(buf1.Bytes()[1] >= byte(100), check.Equals, true)
}
