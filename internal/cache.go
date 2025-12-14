package internal

import (
	"encoding/base64"
	"net/http"
	"time"
)

type Cache interface {
	Get(key string) (*CacheEntity, error)
	Set(key string, value *CacheEntity) error
}

type CacheEntity struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	ExpiresAt  time.Time
}

func CacheMiddleware(cache Cache) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			etag := setEtagHeader(w, r)
			cached, _ := cache.Get(etag)
			if cached != nil {
				w.Header().Set("X-Cache-Status", "HIT")
				w.WriteHeader(cached.StatusCode)
				copyHeaders(w.Header(), cached.Header)
				w.Write(cached.Body)
				return
			}

			next.ServeHTTP(w, r)

			cache.Set(etag, nil)
			w.Header().Set("X-Cache-Status", "MISS")
		})
	}
}

func setEtagHeader(w http.ResponseWriter, r *http.Request) string {
	etag := base64.StdEncoding.EncodeToString([]byte(r.Method + ":" + r.URL.String()))
	w.Header().Set("Etag", etag)
	return etag
}
