package fakehttp

import (
	"bufio"
	"net"
	"net/http"
)

type FakeTransport struct {
	Dial func(hostname, network string) (net.Conn, error)
}

func (t *FakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clientConn, err := t.Dial("", "")
	if err != nil {
		return nil, err
	}
	err = req.Write(clientConn)
	if err != nil {
		return nil, err
	}
	return http.ReadResponse(bufio.NewReader(clientConn), req)
}
