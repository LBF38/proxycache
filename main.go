package main

import (
	"log"
	"net/http"

	"github.com/LBF38/proxycache/internal"
)

func main() {
	log.Println("WIP")

	origin := "http://127.0.0.1:8080"
	proxy := &internal.Proxy{OriginServer: origin}

	if err := http.ListenAndServe(":5000", proxy); err != nil {
		log.Fatalf("error starting server, %v", err)
	}
}
