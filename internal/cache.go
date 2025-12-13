package internal

import (
	"net/http"
	"time"
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
}

type CacheEntity struct {
	Value      http.Response
	Expiration time.Duration
}
