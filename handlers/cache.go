package handlers

import (
	"fmt"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/vegardvb/cesium-terrain-server/log"
	"net/http"
	"net/url"
)

type Cache struct {
	mc      *memcache.Client
	handler http.Handler
	Limit   Bytes
	limiter LimiterFactory
}

func NewCache(connstr string, handler http.Handler, limit Bytes, limiter LimiterFactory) http.Handler {
	return &Cache{
		mc:      memcache.New(connstr),
		handler: handler,
		Limit:   limit,
		limiter: limiter,
	}
}

func (this *Cache) generateKey(r *http.Request) string {
	if key, ok := r.Header["X-Memcache-Key"]; ok {
		return key[0]
	}

	// Use the request URI as a key.
	url, _ := url.Parse(r.URL.String())
	return url.RequestURI()
}

func (this *Cache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var limiter ResponseLimiter
	var recorder http.ResponseWriter
	rec := NewRecorder()

	// If a limiter is provided, wrap the recorder with it.
	if this.limiter != nil {
		limiter = this.limiter(rec, this.Limit)
		recorder = limiter
	} else {
		recorder = rec
	}

	// Write to both the recorder and original writer.
	tee := MultiWriter(w, recorder)
	this.handler.ServeHTTP(tee, r)

	// Only cache 200 responses.
	if rec.Code != 200 {
		return
	}

	// If the cache limit has been exceeded, don't proceed to cache the
	// response.
	if limiter != nil && limiter.LimitExceeded() {
		log.Debug(fmt.Sprintf("cache limit exceeded for %s", r.URL.String()))
		return
	}

	// Cache the response.
	key := this.generateKey(r)
	log.Debug(fmt.Sprintf("setting key: %s", key))
	if err := this.mc.Set(&memcache.Item{Key: key, Value: rec.Body.Bytes()}); err != nil {
		log.Err(err.Error())
	}

	return
}
