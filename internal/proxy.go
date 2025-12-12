package internal

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Proxy struct {
	OriginServer string
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
			log.Printf("error remote addr %v", err)
			return
		}
		r.Header.Set("X-Forwarded-For", h)
		r.Header.Set("X-Real-Ip", h)
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("error requesting server: %v", err)
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}

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
	close(done)
}
