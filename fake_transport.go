package fakehttp

import (
	"bufio"
	"net"
	"net/http"
)

type FakeTransport struct {
	serverListener *FakeListener
}

func (t *FakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clientConn, serverConn := NewFakeConnPair(4)
	err := req.Write(clientConn)
	if err != nil {
		return nil, err
	}
	select {
	case t.serverListener.connC <- serverConn:
	case <-t.serverListener.closeC:
	default:
		return nil, net.ErrClosed
	}
	return http.ReadResponse(bufio.NewReader(clientConn), req)
}
