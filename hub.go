package fakehttp

import (
	"context"
	"net"
	"net/http"
	"sync"
)

// Hub is a bridge between http client and server.
// Use [*Hub.Listener] to create a fake listener, and [*Hub.Transport] to create a fake transport.
type Hub struct {
	mu        sync.Mutex
	listener  *FakeListener
	transport *FakeTransport
}

func NewHub() *Hub {
	return &Hub{
		listener: NewFakeListener(4),
	}
}

func (hub *Hub) Listener() net.Listener {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	return hub.listener
}

func (hub *Hub) Transport() http.RoundTripper {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if hub.transport == nil {
		hub.transport = &FakeTransport{Dial: hub.Dial}
	}
	return hub.transport
}

func (hub *Hub) HTTPClient() *http.Client { return &http.Client{Transport: hub.Transport()} }

func (hub *Hub) Dial(_, _ string) (net.Conn, error) {
	return hub.DialContext(context.Background(), "", "")
}

func (hub *Hub) DialContext(_ context.Context, _, _ string) (net.Conn, error) {
	clientConn, serverConn := NewFakeConnPair(4)
	select {
	case hub.listener.connC <- serverConn:
	case <-hub.listener.closeC:
		return nil, net.ErrClosed
	}
	return clientConn, nil
}
