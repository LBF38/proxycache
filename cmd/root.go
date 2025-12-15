/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/LBF38/proxycache/internal"
	"github.com/spf13/cobra"
)

var port int
var host string
var origin string

var rootCmd = &cobra.Command{
	Use:   "proxycache",
	Short: "A simple HTTP reverse proxy with caching",
	Long: `A simple HTTP reverse proxy with caching.
This is a challenge to build an HTTP reverse proxy from scratch in Go and with caching`,
	Run: func(cmd *cobra.Command, args []string) {
		if port < 1 || port > 65535 {
			fmt.Fprintf(os.Stderr, "Error: port must be between 1 and 65535\n")
			os.Exit(1)
		}

		cache := internal.NewInMemoryCache(1024 * 1024)
		proxy := internal.NewProxy(origin, internal.WithMiddlewares(internal.CacheMiddleware(cache)))

		log.Printf("Proxy listening on %s:%d", host, port)
		if err := http.ListenAndServe(net.JoinHostPort(host, strconv.Itoa(port)), proxy); err != nil {
			log.Fatalf("error starting proxy, %v", err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().IntVarP(&port, "port", "p", 5000, "Port to expose the proxy")
	rootCmd.Flags().StringVarP(&host, "host", "H", "localhost", "Host for the proxy")
	rootCmd.Flags().StringVarP(&origin, "origin", "O", "http://localhost:8000", "Origin server to proxy")
}
