package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

func main() {
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if err := http.ListenAndServe(":8001", server); err != nil {
			log.Fatalf("error second server: %v", err)
		}
	}()

	wg.Wait()
}
