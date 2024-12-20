package fakehttp

import (
	"net"
	"sync"
)

var _ net.Listener = (*FakeListener)(nil)

type FakeListener struct {
	connC     chan *FakeConn
	closeOnce sync.Once
	closeC    chan struct{}
}

func NewFakeListener(maxConnWaiting int) *FakeListener {
	return &FakeListener{
		connC:  make(chan *FakeConn, maxConnWaiting),
		closeC: make(chan struct{}),
	}
}

func (l *FakeListener) Accept() (net.Conn, error) {
	select {
	case <-l.closeC:
	case conn, ok := <-l.connC:
		if ok {
			return conn, nil
		}
	}
	return nil, net.ErrClosed
}

var fakeAddr net.Addr = &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}

func (l *FakeListener) Addr() net.Addr { return fakeAddr }

func (l *FakeListener) Close() error {
	l.closeOnce.Do(func() { close(l.closeC) })
	return nil
}
