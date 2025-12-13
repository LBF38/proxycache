package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

func main() {
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router := http.NewServeMux()

		router.Handle("/trailer", trailer())
		router.Handle("/stream", stream())
		router.Handle("/data", data())
		router.Handle("/", defaultHandler())

		router.ServeHTTP(w, r)
	})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Println("Listening on port 8000")
		if err := http.ListenAndServe(":8000", server); err != nil {
			log.Fatalf("error with HTTP server: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		log.Println("Listening on port 8001")
		if err := http.ListenAndServeTLS(":8001", "cert.pem", "key.pem", server); err != nil {
			log.Fatalf("error second server: %v", err)
		}
	}()

	wg.Wait()
}

func trailer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Trailer", "X-Trailer,X-random")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "body content")
		w.Header().Set("X-Trailer", "Value")
		w.Header().Set("X-random", "more things")
	}
}

func stream() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Begin")
		flusher.Flush()
		time.Sleep(1 * time.Second)
		fmt.Fprintf(w, "End")
		flusher.Flush()
	}
}

func data() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		data := map[string]any{
			"key": "value",
			"random": map[string]int{
				"data": 1,
			},
		}
		json.NewEncoder(w).Encode(&data)
	}
}

func defaultHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Welcome to the demo !\n ")
		fmt.Fprintln(w, "RemoteAddr:", r.RemoteAddr)
		fmt.Fprintln(w, r.Method, r.URL.Path, r.Proto)
		fmt.Fprintln(w, "Host:", r.Host)
		for k, h := range r.Header {
			for _, v := range h {
				fmt.Fprintln(w, k+":", v)
			}
		}
	}
}
