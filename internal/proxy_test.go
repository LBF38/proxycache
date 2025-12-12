package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	t.Run("check status code, body & headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "text/plain")
			w.Header().Set("X-testing-header", "some value")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "some test")
		}))
		defer server.Close()
		proxy := &Proxy{server.URL}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, req)

		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, "text/plain", response.Header().Get("content-type"))
		require.Equal(t, "some value", response.Header().Get("X-testing-header"))
		require.Equal(t, "some test", response.Body.String())
	})

	t.Run("X-Forwarded-For", func(t *testing.T) {
		var header string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header = r.Header.Get("X-Forwarded-For")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "body content")
		}))
		defer server.Close()
		proxy := &Proxy{server.URL}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		req.RemoteAddr = "10.0.0.1:45"
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, req)

		require.Equal(t, "10.0.0.1", header)
	})

	t.Run("stream support", func(t *testing.T) {
		// t.Skip("some workarounds")
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		}))
		defer server.Close()
		proxy := &Proxy{server.URL}
		req := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, req)

		assert.True(t, response.Flushed)
		assert.Equal(t, []string{"some content", "more content"}, strings.Split(strings.Trim(response.Body.String(), "\n"), "\n"))
	})
}
