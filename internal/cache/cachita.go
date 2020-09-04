package cache

import (
	"context"
	"time"

	"github.com/gadelkareem/cachita"
	"github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
)

// CachitaCache implements Cache with `gadelkareem/cachita` package.
type CachitaCache struct {
	cache cachita.Cache
}

func NewCachitaCache() *CachitaCache {
	return &CachitaCache{cache: cachita.Memory()}
}

// Get reads an object data from the cache.
func (c *CachitaCache) GetAlbum(ctx context.Context, key string) (*photoslibrary.Album, error) {
	i := &photoslibrary.Album{}
	err := c.cache.Get(c.encodeAlbumKey(key), i)
	if err == cachita.ErrNotFound {
		return nil, ErrCacheMiss
	}

	return i, err
}

// Put store an object data to the cache.
func (c *CachitaCache) PutAlbum(ctx context.Context, key string, album *photoslibrary.Album, ttl time.Duration) error {
	return c.cache.Put(c.encodeAlbumKey(key), *album, ttl)
}

// InvalidateAlbum removes the specified Album from the cache.
func (c *CachitaCache) InvalidateAlbum(ctx context.Context, key string) error {
	return c.cache.Invalidate(c.encodeAlbumKey(key))
}

// encodeAlbumKey returns the cache key for an Album.
func (c *CachitaCache) encodeAlbumKey(key string) string {
	return "album:" + key
}