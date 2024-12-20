package fakehttp_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/eudierfisher/fakehttp"
)

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}

func TestHelloWorld(t *testing.T) {
	// init HTTP server
	http.Handle("/hello", http.HandlerFunc(helloWorld))
	server := &http.Server{Addr: ":8080"}
	defer func() {
		t.Log("Shutting down server")
		server.Close()
	}()

	// init hub and HTTP client
	hub := fakehttp.NewHub()
	client := &http.Client{Transport: hub.Transport()}
	// or just use
	// client := hub.HTTPClient()

	go func() {
		t.Log("Server is pretending to listen on 127.0.0.1:8080")
		server.Serve(hub.Listener())
	}()

	resp, err := client.Get("http://127.0.0.1:8080/hello")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	b.ReadFrom(resp.Body)
	t.Log(b.String())
}

func newSSEClient(r io.Reader) *SSEClient {
	sseClient := &SSEClient{newLine: LF, scanner: bufio.NewScanner(r)}
	sseClient.scanner.Split(sseClient.Split)
	return sseClient
}

const (
	LF   = "\n"
	CRLF = "\r\n"
)

func (c *SSEClient) SetNewLine(newLine string) {
	if newLine != LF && newLine != CRLF {
		return
	}
	c.newLine = newLine
}

func (c *SSEClient) Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, c.newLine+c.newLine); i >= 0 {
		// We have a full newline-terminated line.
		return i + 2*len(c.newLine), data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func (c *SSEClient) ReadEvent() ([]byte, error) {
	if c.scanner.Scan() {
		event := c.scanner.Bytes()
		return event, nil
	}
	if err := c.scanner.Err(); err != context.Canceled {
		return nil, err
	}
	return nil, io.EOF
}

type SSEClient struct {
	newLine string
	scanner *bufio.Scanner
}

func httpsse(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ticker := time.NewTicker(1 * time.Second)
	for i := 0; i < 8; i++ {
		// Simulate sending events every second
		fmt.Fprintf(w, "data: %s\n\n", time.Now().Format(time.Stamp))
		w.(http.Flusher).Flush()
		<-ticker.C
	}
	ticker.Stop()
}

func TestSSE(t *testing.T) {
	// init HTTP server
	http.Handle("/httpsse", http.HandlerFunc(httpsse))
	server := &http.Server{Addr: ":8080"}
	defer func() {
		t.Log("Shutting down server")
		server.Close()
	}()

	// init hub and HTTP client
	hub := fakehttp.NewHub()
	client := &http.Client{Transport: hub.Transport()}
	// or just use
	// client := hub.HTTPClient()

	go func() {
		t.Log("Server is pretending to listen on 127.0.0.1:8080")
		server.Serve(hub.Listener())
	}()

	resp, err := client.Get("http://127.0.0.1:8080/httpsse")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer resp.Body.Close()

	// The SSE client only splits response body into events without further parsing.
	// Codes are borrowed from [r3labs/sse](https://github.com/r3labs/sse).
	sseClient := newSSEClient(resp.Body)
	for i := 0; i < 8; i++ {
		e, err := sseClient.ReadEvent()
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log(string(e))
	}
}
