package main

import (
	"log"
	"net/http"

	"github.com/LBF38/proxycache/internal"
)

func main() {
	log.Println("WIP")

	origin := "http://127.0.0.1:8000"
	cache := internal.NewInMemoryCache(1024 * 1024)
	proxy := internal.NewProxy(origin, internal.WithMiddlewares(internal.CacheMiddleware(cache)))

	log.Println("Listening on port 5000")
	if err := http.ListenAndServe(":5000", proxy); err != nil {
		log.Fatalf("error starting server, %v", err)
	}
}
