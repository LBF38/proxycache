package internal

import (
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
	log.Printf("request %v", r.RequestURI)
	r.RequestURI = ""
	log.Println(r)
	resp, err := http.DefaultClient.Do(r)
	log.Println(resp)
	w.WriteHeader(resp.StatusCode)
}
