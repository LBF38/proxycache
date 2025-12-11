package internal

import (
	"io"
	"log"
	"net/http"
	"net/url"
)

type Proxy struct {
	OriginServer string
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin, err := url.Parse(p.OriginServer)
	if err != nil {
		log.Fatalf("error parsing url, got %v", err)
	}

	r.Host = origin.Host
	r.URL.Host = origin.Host
	r.URL.Scheme = origin.Scheme
	r.RequestURI = ""
	resp, err := http.DefaultClient.Do(r)

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
