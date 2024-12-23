package fakehttp

import (
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var _ net.Conn = (*FakeConn)(nil)

func NewFakeConnPair(maxDataCount int) (a, b *FakeConn) {
	a2b, b2a := NewFakeChannel(maxDataCount), NewFakeChannel(maxDataCount)
	return &FakeConn{b2a, a2b}, &FakeConn{a2b, b2a}
}

type FakeConn struct {
	ReadChannel  *FakeChannel
	WriteChannel *FakeChannel
}

func (c *FakeConn) Close() error {
	c.ReadChannel.Close()
	c.WriteChannel.Close()
	return nil
}

func (c *FakeConn) LocalAddr() net.Addr { return fakeAddr }

func (c *FakeConn) RemoteAddr() net.Addr { return fakeAddr }

func (c *FakeConn) Read(b []byte) (int, error) { return c.ReadChannel.Read(b) }

func (c *FakeConn) Write(b []byte) (int, error) { return c.WriteChannel.Write(b) }

func (c *FakeConn) SetReadDeadline(t time.Time) error {
	if t.IsZero() {
		log.Println("clear read deadline")
	} else {
		log.Println("set read deadline", t.Format(time.Stamp))
	}
	return c.ReadChannel.SetReadDeadline(t)
}
func (c *FakeConn) SetWriteDeadline(t time.Time) error { return c.WriteChannel.SetWriteDeadline(t) }
func (c *FakeConn) SetDeadline(t time.Time) error {
	c.SetReadDeadline(t)
	c.SetWriteDeadline(t)
	return nil
}

func NewFakeChannel(maxDataCount int) *FakeChannel {
	return &FakeChannel{
		readDeadlineC: make(chan struct{}),
		closeC:        make(chan struct{}),
		dataC:         make(chan []byte, maxDataCount),
	}
}

type FakeChannel struct {
	mu            sync.Mutex
	readDeadline  time.Time
	writeDeadline time.Time
	readDeadlineC chan struct{}

	closeOnce sync.Once
	closeC    chan struct{}
	dataC     chan []byte
}

func (c *FakeChannel) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t == c.readDeadline {
		return nil
	}
	c.readDeadline = t
	select {
	case c.readDeadlineC <- struct{}{}:
		// notify Read
	default:
	}
	return nil
}

func (c *FakeChannel) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeDeadline = t
	return nil
}

var noDeadlineC = make(chan time.Time, 1)

func (c *FakeChannel) Read(b []byte) (n int, err error) {
	c.mu.Lock()
	readDeadline := c.readDeadline
	c.mu.Unlock()

	if !readDeadline.IsZero() && readDeadline.Before(time.Now()) {
		// fast check and fail
		return 0, os.ErrDeadlineExceeded
	}

	// the check above makes sure that when the loop starts,
	// the read deadline is always valid (0 or in the future);
	// when the following loop blocks, it must indicate that channel dataC has been emptied
	var ch <-chan time.Time = noDeadlineC
	var timer *time.Timer
	for loop := true; loop; {
		select {
		case dl := <-ch:
			// entering this case means that the timer is not nil and has expired
			log.Println("timer expired at", dl.Local().String())
			timer = nil
			n, err = 0, os.ErrDeadlineExceeded
			loop = false
			continue
		case <-c.readDeadlineC:
			// someone has called SetReadDeadline
			if timer != nil {
				timer.Stop()
			}

			c.mu.Lock()
			readDeadline = c.readDeadline
			c.mu.Unlock()
			if readDeadline.IsZero() {
				ch = noDeadlineC
				continue
			}
			if readDeadline.Before(time.Now()) {
				// deadline exceeded, cancel Read
				n, err = 0, os.ErrDeadlineExceeded
				loop = false
				continue
			}
			timer = time.NewTimer(time.Until(readDeadline))
			ch = timer.C
		case <-c.closeC:
			n, err = 0, io.EOF
			loop = false
		case data, ok := <-c.dataC:
			if ok && len(data) <= len(b) {
				n, err = copy(b, data), nil
			}
			loop = false
		}
	}
	if timer != nil {
		timer.Stop()
	}
	return
}

func (c *FakeChannel) Write(b []byte) (int, error) {
	select {
	case <-c.closeC:
		return 0, net.ErrClosed
	case c.dataC <- b:
		return len(b), nil
	}
}

func (c *FakeChannel) Close() error {
	c.closeOnce.Do(func() { close(c.closeC) })
	return nil
}

func (c *FakeChannel) Closed() bool {
	select {
	case <-c.closeC:
		return true
	default:
	}
	return false
}
