package internal

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type StubCache struct {
	store    map[string]*CacheEntity
	getCalls int
	getError error
	setCalls int
	setError error
}

func newStubCache(store map[string]*CacheEntity, get, set error) *StubCache {
	if store == nil {
		store = map[string]*CacheEntity{}
	}
	return &StubCache{store, 0, get, 0, set}
}

func (c *StubCache) Get(key string) (*CacheEntity, error) {
	c.getCalls++
	return c.store[key], c.getError
}

func (c *StubCache) Set(key string, value *CacheEntity) error {
	c.setCalls++
	c.store[key] = value
	return c.setError
}

func TestCache(t *testing.T) {
	t.Run("GET request cached on first call", func(t *testing.T) {
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "first response")
		})
		defer server.Close()
		cache := newStubCache(nil, errors.New("not found"), nil)
		cacheMiddleware := CacheMiddleware(cache)
		proxy := NewProxy(server.URL, WithMiddlewares(cacheMiddleware))
		request := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()
		expected := &CacheEntity{
			StatusCode: 200,
			Body:       []byte("first response"),
		}

		proxy.ServeHTTP(response, request)

		assert.Equal(t, "MISS", response.Header().Get("X-Cache-Status"))
		decodedETag := getDecodedEtag(t, response)
		assert.Equal(t, http.MethodGet+":"+request.URL.String(), decodedETag)
		assert.Equal(t, 1, cache.getCalls)
		assert.Equal(t, 1, cache.setCalls)
		cached := cache.store[response.Header().Get("ETag")]
		require.NotNil(t, cached) // TODO, WIP
		assert.Equal(t, expected.StatusCode, cached.StatusCode)
		assert.Equal(t, string(expected.Body), string(cached.Body))
		assert.NotEmpty(t, cached.Header) // TODO
	})

	t.Run("return the cached response", func(t *testing.T) {
		server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "real response")
		})
		defer server.Close()
		request := httptest.NewRequest(http.MethodGet, server.URL, nil)
		response := httptest.NewRecorder()
		store := map[string]*CacheEntity{
			buildEtag(t, request): {
				StatusCode: 200,
				Header:     http.Header{},
				Body:       []byte("cached response"),
				ExpiresAt:  time.Now().Add(time.Minute),
			},
		}
		cache := newStubCache(store, nil, nil)
		proxy := NewProxy(server.URL, WithMiddlewares(CacheMiddleware(cache)))

		proxy.ServeHTTP(response, request)

		assert.Equal(t, "HIT", response.Header().Get("X-Cache-Status"))
		assert.Equal(t, 1, cache.getCalls)
		assert.Equal(t, 0, cache.setCalls)
		assert.Equal(t, "cached response", response.Body.String())
	})
}

func getDecodedEtag(t testing.TB, response *httptest.ResponseRecorder) string {
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

func TestNoCache(t *testing.T) {
	tests := []struct {
		desc            string
		method          string
		requestHeaders  http.Header
		responseHeaders http.Header
		body            io.Reader
		wantCached      bool
	}{
		{
			desc:   "Cache-Control: no-cache",
			method: http.MethodGet,
			requestHeaders: http.Header{
				"Cache-Control": {"no-cache"},
			},
			responseHeaders: http.Header{
				"Cache-Control": {"no-cache"},
			},
		},
		{
			desc:   "Cache-Control: no-store",
			method: http.MethodGet,
			requestHeaders: http.Header{
				"Cache-Control": {"no-store"},
			},
			responseHeaders: http.Header{
				"Cache-Control": {"no-store"},
			},
		},
		{
			desc:   "Cache-Control: private",
			method: http.MethodGet,
			requestHeaders: http.Header{
				"Cache-Control": {"private"},
			},
			responseHeaders: http.Header{
				"Cache-Control": {"private"},
			},
		},
		{
			desc:   "Cache-Control: public",
			method: http.MethodGet,
			requestHeaders: http.Header{
				"Cache-Control": {"public"},
			},
			responseHeaders: http.Header{
				"Cache-Control": {"public"},
			},
			wantCached: true,
		},
		{
			desc:   "Cache-Control: max-ages",
			method: http.MethodGet,
			requestHeaders: http.Header{
				"Cache-Control": {"max-age=86400"},
			},
			responseHeaders: http.Header{
				"Cache-Control": {"max-age=86400"},
			},
			wantCached: true,
		},
		{
			desc:   "Cache-Control: public,max-ages",
			method: http.MethodGet,
			requestHeaders: http.Header{
				"Cache-Control": {"public", "max-age=86400"},
			},
			responseHeaders: http.Header{
				"Cache-Control": {"public", "max-age=86400"},
			},
			wantCached: true,
		},
		{
			desc:   "Cache-Control: private,max-ages",
			method: http.MethodGet,
			requestHeaders: http.Header{
				"Cache-Control": {"private", "max-age=86400"},
			},
			responseHeaders: http.Header{
				"Cache-Control": {"private", "max-age=86400"},
			},
			wantCached: false,
		},
		{
			desc:       "Cache for GET",
			method:     http.MethodGet,
			wantCached: true,
		},
		{
			desc:       "Cache for HEAD",
			method:     http.MethodHead,
			wantCached: true,
		},
		{
			desc:       "No cache for OPTIONS",
			method:     http.MethodOptions,
			wantCached: false,
		},
		{
			desc:       "No cache for TRACE",
			method:     http.MethodTrace,
			wantCached: false,
		},
		{
			desc:       "No cache for POST",
			method:     http.MethodPost,
			wantCached: false,
		},
		{
			desc:       "No cache for PUT",
			method:     http.MethodPut,
			wantCached: false,
		},
		{
			desc:       "No cache for PATCH",
			method:     http.MethodPatch,
			wantCached: false,
		},
		{
			desc:       "No cache for DELETE",
			method:     http.MethodDelete,
			wantCached: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
				addHeaders(w.Header(), tt.responseHeaders)
			})
			defer server.Close()
			cache := newStubCache(map[string]*CacheEntity{}, nil, nil)
			proxy := NewProxy(server.URL, WithMiddlewares(CacheMiddleware(cache)))
			response := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, server.URL, tt.body)
			addHeaders(request.Header, tt.requestHeaders)

			proxy.ServeHTTP(response, request)

			if tt.wantCached {
				assert.Equal(t, 1, cache.getCalls, "cache get calls")
				assert.Equal(t, 1, cache.setCalls, "cache set calls")
				assert.NotEmpty(t, response.Header().Get("ETag"))
				assert.Equal(t, "MISS", response.Header().Get("X-Cache-Status"))
				return
			}
			assert.Equal(t, 0, cache.getCalls, "cache get calls")
			assert.Equal(t, 0, cache.setCalls, "cache set calls")
			assert.Empty(t, response.Header().Get("ETag"))
			assert.Equal(t, "BYPASS", response.Header().Get("X-Cache-Status"))
		})
	}
}
