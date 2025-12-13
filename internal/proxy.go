package internal

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Proxy struct {
	OriginServer string // TODO: change to a Config
	cache        Cache
}

const (
	HeaderForwardedHost   = "X-Forwarded-Host"
	HeaderForwardedPort   = "X-Forwarded-Port"
	HeaderForwardedProto  = "X-Forwarded-Proto"
	HeaderForwardedServer = "X-Forwarded-Server"
)

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin, err := url.Parse(p.OriginServer)
	if err != nil {
		log.Fatalf("error parsing url, got %v", err)
	}

	err = p.updateRequest(r, origin, w)
	if err != nil {
		log.Printf("error updating request, got %v", err)
	}

	_, _ = p.cache.Get("")

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("error requesting server: %v", err)
		return
	}

	copyHeaders(w.Header(), resp.Header)

	p.cache.Set("", []byte(""))
	w.Header().Set("X-Cache-Status", "MISS")

	var trailerKeys []string
	for key := range resp.Trailer {
		trailerKeys = append(trailerKeys, key)
	}
	w.Header().Set("X-Trailer", strings.Join(trailerKeys, ","))

	// for streaming connections/data
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-time.Tick(1 * time.Millisecond):
				w.(http.Flusher).Flush()
			case <-done:
				return
			}
		}
	}()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	copyHeaders(w.Header(), resp.Trailer)

	close(done)
}

func (p *Proxy) updateRequest(r *http.Request, origin *url.URL, w http.ResponseWriter) error {
	r.Header.Set(HeaderForwardedHost, r.Host)
	r.Header.Set(HeaderForwardedPort, r.URL.Port())
	r.Header.Set(HeaderForwardedProto, r.Proto)
	r.Header.Set(HeaderForwardedServer, "ProxyCache") // WIP

	r.Host = origin.Host
	r.URL.Host = origin.Host
	r.URL.Scheme = origin.Scheme
	r.RequestURI = ""
	if r.RemoteAddr != "" {
		h, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return fmt.Errorf("error reading address: %w", err)
		}
		r.Header.Set("X-Forwarded-For", h)
		r.Header.Set("X-Real-Ip", h)
	}
	if r.UserAgent() == "" {
		r.Header.Set("User-Agent", "")
	}
	return nil
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Set(key, value)
		}
	}
}
