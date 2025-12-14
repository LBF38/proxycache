package internal

import (
	"bytes"
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

			rec := &responseRecorder{ResponseWriter: w, body: bytes.NewBuffer(nil)}
			next.ServeHTTP(rec, r)

			entity := &CacheEntity{
				StatusCode: rec.statusCode,
				Header:     rec.Header(),
				Body:       rec.body.Bytes(),
			}
			cache.Set(etag, entity)
			w.Header().Set("X-Cache-Status", "MISS")
		})
	}
}

func setEtagHeader(w http.ResponseWriter, r *http.Request) string {
	etag := base64.StdEncoding.EncodeToString([]byte(r.Method + ":" + r.URL.String()))
	w.Header().Set("Etag", etag)
	return etag
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
