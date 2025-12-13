package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type StubCache struct {
	getCalls int
	setCalls int
}

func (c *StubCache) Get(key string) ([]byte, error) {
	c.getCalls++
	return nil, nil
}

func (c *StubCache) Set(key string, value []byte) error {
	c.setCalls++
	return nil
}

func TestCache(t *testing.T) {
	t.Run("should cache the GET request", func(t *testing.T) {
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "cached response")
		})
		defer server.Close()
		cache := &StubCache{}
		proxy := NewProxy(server.URL, cache)
		request := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, request)

		assert.Equal(t, "MISS", response.Header().Get("X-Cache-Status"))
		assert.Equal(t, 1, cache.getCalls)
		assert.Equal(t, 1, cache.setCalls)
	})
}
