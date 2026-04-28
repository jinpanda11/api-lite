package model

import (
	"sync"
	"time"
)

type tokenCacheEntry struct {
	token    *Token
	cachedAt time.Time
}

var (
	tokenCache   = map[string]*tokenCacheEntry{}
	tokenCacheMu sync.RWMutex
)

func init() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			tokenCacheMu.Lock()
			now := time.Now()
			for k, v := range tokenCache {
				if now.Sub(v.cachedAt) > 10*time.Minute {
					delete(tokenCache, k)
				}
			}
			tokenCacheMu.Unlock()
		}
	}()
}

// CacheGetTokenByKey returns a cached token if available, otherwise queries DB.
func CacheGetTokenByKey(key string) (*Token, error) {
	tokenCacheMu.RLock()
	entry, ok := tokenCache[key]
	tokenCacheMu.RUnlock()
	if ok && time.Since(entry.cachedAt) < 10*time.Minute {
		return entry.token, nil
	}

	token, err := GetTokenByKey(key)
	if err != nil {
		return nil, err
	}

	tokenCacheMu.Lock()
	tokenCache[key] = &tokenCacheEntry{token: token, cachedAt: time.Now()}
	tokenCacheMu.Unlock()

	return token, nil
}

// CacheInvalidateToken removes a token from the cache by key.
func CacheInvalidateToken(key string) {
	tokenCacheMu.Lock()
	delete(tokenCache, key)
	tokenCacheMu.Unlock()
}
