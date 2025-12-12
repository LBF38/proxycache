package internal

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	t.Run("check status code, body & headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "text/plain")
			w.Header().Set("X-testing-header", "some value")
			fmt.Fprintf(w, "some test")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		proxy := &Proxy{server.URL}
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
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
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		req.RemoteAddr = "10.0.0.1:45"
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, req)

		require.Equal(t, "10.0.0.1", header)
		log.Println(response.Body.String())
	})
}
