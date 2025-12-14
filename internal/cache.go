package internal

import (
	"net/http"
	"time"
)

type Cache interface {
	Get(key string) (CacheEntity, error)
	Set(key string, value CacheEntity) error
}

type CacheEntity struct {
	Value      http.Response
	Expiration time.Duration
}
