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
	origin      *url.URL // TODO: change to a Config
	middlewares []Middleware
	http.Handler
}

type ProxyOptions func(*Proxy)

type Middleware func(http.Handler) http.Handler

func chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

const (
	HeaderForwardedHost   = "X-Forwarded-Host"
	HeaderForwardedPort   = "X-Forwarded-Port"
	HeaderForwardedProto  = "X-Forwarded-Proto"
	HeaderForwardedServer = "X-Forwarded-Server"
)

func NewProxy(origin string, options ...ProxyOptions) *Proxy {
	proxy := new(Proxy)
	o, err := url.Parse(origin)
	if err != nil {
		log.Fatalf("error parsing url, got %v", err)
	}
	proxy.origin = o

	for _, option := range options {
		option(proxy)
	}

	proxy.Handler = chain(proxy.middlewares...)(proxy.callServer())

	return proxy
}

func WithMiddlewares(middlewares ...Middleware) ProxyOptions {
	return func(p *Proxy) {
		p.middlewares = append(p.middlewares, middlewares...)
	}
}

func (p *Proxy) callServer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := p.updateRequest(r, p.origin, w)
		if err != nil {
			log.Printf("error updating request, got %v", err)
		}

		log.Printf("request: %s %s %s", r.Method, r.URL.String(), r.Proto)
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("error requesting server: %v", err)
			return
		}

		addHeaders(w.Header(), resp.Header)

		var trailerKeys []string
		for key := range resp.Trailer {
			trailerKeys = append(trailerKeys, key)
		}
		if len(trailerKeys) > 0 {
			w.Header().Set("X-Trailer", strings.Join(trailerKeys, ","))
		}

		// for streaming connections/data
		done := p.flush(w)

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)

		if len(trailerKeys) > 0 {
			setHeaders(w.Header(), resp.Trailer)
		}

		close(done)
	}
}

func (*Proxy) flush(w http.ResponseWriter) chan bool {
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
	return done
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

func setHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Set(key, value)
		}
	}
}

func addHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
