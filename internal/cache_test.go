package internal

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type StubCache struct {
	store    map[string]CacheEntity
	getCalls int
	getError error
	setCalls int
	setError error
}

func newStubCache(store map[string]CacheEntity, get, set error) *StubCache {
	if store == nil {
		store = map[string]CacheEntity{}
	}
	return &StubCache{store, 0, get, 0, set}
}

func (c *StubCache) Get(key string) (CacheEntity, error) {
	c.getCalls++
	return c.store[key], c.getError
}

func (c *StubCache) Set(key string, value CacheEntity) error {
	c.setCalls++
	c.store[key] = value
	return c.setError
}

func TestCache(t *testing.T) {
	t.Run("GET request cached on first call", func(t *testing.T) {
		t.Skip("WIP")
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "first response")
		})
		defer server.Close()
		cache := newStubCache(nil, errors.New("not found"), nil)
		proxy := NewProxy(server.URL, cache)
		request := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()

		proxy.ServeHTTP(response, request)

		assert.Equal(t, "MISS", response.Header().Get("X-Cache-Status"))
		etag := getEtag(t, response)
		assert.Equal(t, http.MethodGet+":"+request.URL.String(), etag)
		assert.Equal(t, 1, cache.getCalls)
		assert.Equal(t, 1, cache.setCalls)
	})

	t.Run("return the cached response", func(t *testing.T) {
		t.Skip("WIP")
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "cached response")
		})
		defer server.Close()
		request := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()
		resp, _ := server.Client().Do(request)
		store := map[string]CacheEntity{
			buildEtag(t, request): {
				Value:      *resp,
				Expiration: 0,
			},
		}
		cache := newStubCache(store, nil, nil)
		proxy := NewProxy(server.URL, cache)
		// cached := httptest.NewRecorder()

		proxy.ServeHTTP(response, request)
		proxy.ServeHTTP(response, request)
		// proxy.ServeHTTP(cached, request)

		assert.Equal(t, "HIT", response.Header().Get("X-Cache-Status"))
		assert.Equal(t, 1, cache.getCalls)
		assert.Equal(t, 0, cache.setCalls)
		// assert.Equal(t, "cached response", response.Body.String())
	})
}

func getEtag(t testing.TB, response *httptest.ResponseRecorder) string {
	t.Helper()
	etagBytes, err := base64.StdEncoding.DecodeString(response.Header().Get("ETag"))
	if err != nil {
		t.Error(err)
	}
	etag := string(etagBytes)
	return etag
}

func buildEtag(t testing.TB, r *http.Request) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString([]byte(r.Method + ":" + r.URL.String()))
}
