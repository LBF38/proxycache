package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	t.Run("forward a simple HTTP request - check status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		proxy := &Proxy{server.URL}
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, req)

		require.Equal(t, http.StatusOK, response.Code)
	})
}
