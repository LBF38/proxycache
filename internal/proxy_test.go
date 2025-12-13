package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var cache StubCache

func TestProxy(t *testing.T) {
	t.Run("check status code, body & headers", func(t *testing.T) {
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "text/plain")
			w.Header().Set("X-testing-header", "some value")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "some test")
		})
		defer server.Close()
		proxy := &Proxy{server.URL, &cache}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, req)

		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, "text/plain", response.Header().Get("content-type"))
		assert.Equal(t, "some value", response.Header().Get("X-testing-header"))
		assert.Equal(t, "some test", response.Body.String())
	})

	t.Run("forwarded headers", func(t *testing.T) {
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "hostname", r.Header.Get(HeaderForwardedHost))
			assert.Equal(t, "HTTP/1.1", r.Header.Get(HeaderForwardedProto))
			assert.Equal(t, "ProxyCache", r.Header.Get(HeaderForwardedServer))
			w.Header().Set(HeaderForwardedPort, r.Header.Get(HeaderForwardedPort))
		})
		defer server.Close()
		port, err := url.Parse(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		proxy := &Proxy{server.URL, &cache}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()

		req.Host = "hostname"
		proxy.ServeHTTP(response, req)
		assert.Equal(t, port.Port(), response.Header().Get(HeaderForwardedPort))
	})

	t.Run("Remote addr in headers", func(t *testing.T) {
		var headers []string
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			headers = append(headers, r.Header.Get("X-Forwarded-For"))
			headers = append(headers, r.Header.Get("X-Real-Ip"))
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "body content")
		})
		defer server.Close()
		proxy := &Proxy{server.URL, &cache}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		req.RemoteAddr = "10.0.0.1:45"
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, req)

		assert.Equal(t, 2, len(headers))
		assert.Equal(t, "10.0.0.1", headers[0])
		assert.Equal(t, "10.0.0.1", headers[1])
	})

	t.Run("stream support", func(t *testing.T) {
		// t.Skip("some workarounds")
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "text/event-stream")
			w.Header().Set("Connection", "keep-alive")
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming unsupported", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "some content\n")
			flusher.Flush()
			time.Sleep(time.Millisecond) // Not really great for test...
			fmt.Fprintf(w, "more content\n")
			flusher.Flush()
		})
		defer server.Close()
		proxy := &Proxy{server.URL, &cache}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, req)

		assert.True(t, response.Flushed)
		assert.Equal(t, []string{"some content", "more content"}, strings.Split(strings.Trim(response.Body.String(), "\n"), "\n"))
	})

	t.Run("bad remote addr", func(t *testing.T) {
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		defer server.Close()
		proxy := &Proxy{server.URL, &cache}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		resp := httptest.NewRecorder()

		req.RemoteAddr = "1.2.3.4"
		proxy.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Result().StatusCode)
	})

	t.Run("trailer", func(t *testing.T) {
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Trailer", "X-Trailer,X-random")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "body content")
			w.Header().Set("X-Trailer", "Value")
			w.Header().Set("X-random", "more things")
		})
		defer server.Close()
		proxy := &Proxy{server.URL, &cache}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, req)

		assert.Equal(t, "Value", response.Header().Get("X-Trailer"))
		assert.Equal(t, "more things", response.Header().Get("X-random"))
	})

	t.Run("User-Agent", func(t *testing.T) {
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "tester", r.Header.Get("User-Agent"))
		})
		defer server.Close()
		proxy := &Proxy{server.URL, &cache}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()
		req.Header.Set("user-agent", "tester")

		proxy.ServeHTTP(response, req)
	})

	t.Run("No User-Agent, no forwarding", func(t *testing.T) {
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "", r.Header.Get("User-Agent"))
		})
		defer server.Close()
		proxy := &Proxy{server.URL, &cache}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, req)
	})

	t.Run("HTTP/2", func(t *testing.T) {
		t.Skip("TODO")
	})
	t.Run("HTTP/3 ?", func(t *testing.T) {
		t.Skip("TODO")
	})
	t.Run("websockets ?", func(t *testing.T) {
		t.Skip("TODO")
	})
	t.Run("Basic Authentication", func(t *testing.T) {
		t.Skip("TODO")
	})
	t.Run("OAuth Authentication", func(t *testing.T) {
		t.Skip("TODO")
	})
}

func createTestServer(f http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(f)
}
