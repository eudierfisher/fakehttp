# FakeHTTP

FakeHTTP is a

- memory-based library for http unit tests
- (maybe) good replacement for `httptest`

FakeHTTP is **NOT** a

- debugging tool for browser
- mocking tool generating predefined responses

## Why FakeHTTP

- No network
- Using fully functional `http.Server` and `http.Client`
- HTTP SSE support
- Websocket support

## Library Support

HTTP SSE
|library|version|status|
|-|-|-|
|[r3labs/sse](https://github.com/r3labs/sse)|v2.10.0|✔|

Websocket
|library|version|dial|client|handler|
|-|-|-|-|-|
|[gorilla/websocket](https://github.com/gorilla/websocket)|v1.5.3|✔|✔|✔|
|[golang.org/net/websocket](https://github.com/golang/net)||testing||
|[coder/websocket<br>nhooyr.io/websocket](https://github.com/nhooyr/websocket)||testing||

See [fakehttp-thirdparty-test](https://github.com/eudierfisher/fakehttp-thirdparty-test) repo for more example tests.

## Usage

### Hello World

```go
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
```

Result:
```shell
=== RUN   TestHelloWorld
    d:\go\src\github.com\eudierfisher\fakehttp\http_test.go:36: Server is pretending to listen on 127.0.0.1:8080
    d:\go\src\github.com\eudierfisher\fakehttp\http_test.go:48: Hello World
    d:\go\src\github.com\eudierfisher\fakehttp\http_test.go:25: Shutting down server
--- PASS: TestHelloWorld (0.00s)
PASS
ok      github.com/eudierfisher/fakehttp        5.764s
```

### HTTP SSE

```go
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

	// See http_test.go for more details
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
```

Result:

https://github.com/user-attachments/assets/131d4d4c-9033-4663-a753-8372784411ad

### Websocket

See [fakehttp-thirdparty-test](https://github.com/eudierfisher/fakehttp-thirdparty-test) repo for websocket tests.