package internal

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"slices"
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
			etag := getETag(r)
			if !bypassCacheFromRequest(w, r) {
				cached, _ := cache.Get(etag)
				if cached != nil {
					w.Header().Set("X-Cache-Status", "HIT")
					setEtagHeader(w, etag)
					w.WriteHeader(cached.StatusCode)
					setHeaders(w.Header(), cached.Header)
					w.Write(cached.Body)
					return
				}
			}

			rec := &responseRecorder{ResponseWriter: w, body: bytes.NewBuffer(nil)}
			next.ServeHTTP(rec, r)

			if !bypassCacheFromResponse(rec, r) {
				entity := &CacheEntity{
					StatusCode: rec.statusCode,
					Header:     rec.Header(),
					Body:       rec.body.Bytes(),
				}
				cache.Set(etag, entity)
				setEtagHeader(w, etag)
				w.Header().Set("X-Cache-Status", "MISS")
			}
		})
	}
}

func bypassCacheFromRequest(w http.ResponseWriter, r *http.Request) bool {
	rules := []string{"no-store", "no-cache", "private"}     // bypass
	methodRules := []string{http.MethodGet, http.MethodHead} // allow
	for _, rule := range rules {
		if slices.Contains(r.Header.Values("Cache-Control"), rule) {
			setCacheStatus(w, statusBYPASS)
			return true
		}
	}

	if !slices.Contains(methodRules, r.Method) {
		setCacheStatus(w, statusBYPASS)
		return true
	}
	return false
}

func bypassCacheFromResponse(w http.ResponseWriter, r *http.Request) bool {
	cacheControlRules := []string{"no-store", "no-cache", "private"} // bypass
	methodRules := []string{http.MethodGet, http.MethodHead}         // allow
	// codeRules := []int{200, 203, 204, 206, 300, 301, 308, 404, 405, 410, 414, 501} // allow - ref. RFC9110 15.1
	for _, rule := range cacheControlRules {
		// By default, Cache-Control empty = heuristic caching
		if slices.Contains(w.Header()["Cache-Control"], rule) {
			setCacheStatus(w, statusBYPASS)
			return true
		}
	}

	if !slices.Contains(methodRules, r.Method) {
		setCacheStatus(w, statusBYPASS)
		return true
	}
	return false
}

type cacheStatus string

func (c cacheStatus) String() string {
	return string(c)
}

const (
	statusBYPASS cacheStatus = "BYPASS"
	statusHIT    cacheStatus = "HIT"
	statusMISS   cacheStatus = "MISS"
)

func setCacheStatus(w http.ResponseWriter, status cacheStatus) {
	w.Header().Set("X-Cache-Status", status.String())
}

func setEtagHeader(w http.ResponseWriter, etag string) {
	w.Header().Set("Etag", etag)
}

func getETag(r *http.Request) string {
	return base64.StdEncoding.EncodeToString([]byte(r.Method + ":" + r.URL.String()))
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
