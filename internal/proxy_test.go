package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProxy(t *testing.T) {
	t.Run("check status code, body & headers", func(t *testing.T) {
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "text/plain")
			w.Header().Set("X-testing-header", "some value")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "some test")
		})
		defer server.Close()
		proxy := &Proxy{server.URL}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, req)

		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, "text/plain", response.Header().Get("content-type"))
		assert.Equal(t, "some value", response.Header().Get("X-testing-header"))
		assert.Equal(t, "some test", response.Body.String())
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
		proxy := &Proxy{server.URL}
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
		proxy := &Proxy{server.URL}
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
		proxy := &Proxy{server.URL}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		resp := httptest.NewRecorder()

		req.RemoteAddr = "1.2.3.4"
		proxy.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Result().StatusCode)
	})
}

func createTestServer(f http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(f)
}
