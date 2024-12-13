package fakehttp

import (
	"net"
	"net/http"
)

var _ net.Listener = (*FakeListener)(nil)

type FakeListener struct {
	connC  chan *FakeConn
	closeC chan struct{}
}

func NewFakeListener(maxConnWaiting int) *FakeListener {
	return &FakeListener{
		connC:  make(chan *FakeConn, maxConnWaiting),
		closeC: make(chan struct{}),
	}
}

func (l *FakeListener) Transport() http.RoundTripper {
	return &FakeTransport{serverListener: l}
}

func (l *FakeListener) Client() *http.Client {
	return &http.Client{Transport: l.Transport()}
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
	select {
	case <-l.closeC:
	default:
		close(l.closeC)
	}
	return nil
}
