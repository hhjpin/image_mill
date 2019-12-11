package image_mill

import (
	"sync"
)

type imageCache struct {
	sync.Mutex
	images map[string]string
}

func newImageCache() *imageCache {
	cache := &imageCache{}
	cache.images = map[string]string{}
	return cache
}

func (a *imageCache) Store(key, val string) {
	a.Lock()
	a.images[key] = val
	a.Unlock()
}

func (a *imageCache) Delete(key string) {
	a.Lock()
	delete(a.images, key)
	a.Unlock()
}

func (a *imageCache) Load(key string) string {
	a.Lock()
	data := a.images[key]
	a.Unlock()
	return data
}
