package fakehttp

import (
	"io"
	"net"
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

func (c *FakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *FakeConn) SetWriteDeadline(t time.Time) error { return nil }
func (c *FakeConn) SetDeadline(t time.Time) error      { return nil }

func NewFakeChannel(maxDataCount int) *FakeChannel {
	return &FakeChannel{
		closeC: make(chan struct{}),
		dataC:  make(chan []byte, maxDataCount),
	}
}

type FakeChannel struct {
	closeOnce sync.Once
	closeC    chan struct{}
	dataC     chan []byte
}

func (c *FakeChannel) Read(b []byte) (n int, err error) {
	select {
	case <-c.closeC:
	case data, ok := <-c.dataC:
		if ok && len(data) <= len(b) {
			return copy(b, data), nil
		}
	}
	return 0, io.EOF
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
