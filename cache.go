package gcache

import (
	"sync"
	"time"
)

type Cache interface {
	// Returns item and boolean value for check of key existing
	Get(key string) (*Item, bool)
	// Adds value by key and returns item object.
	Add(key string, value interface{}) *Item
	// Deletes item from cache by key.
	Delete(key string)
	// Returns item. If item doesnt exist, then it adds new value.
	GetOrAdd(key string, value interface{}) *Item
	// Returns true if key exists.
	Contains(key string) bool
	// Returns count of items in the cache
	Count() int
}

type Item struct {
	Value          interface{}
	LastTimeAccess time.Time
}

func NewItem(v interface{}) *Item {
	return &Item{
		Value:          v,
		LastTimeAccess: time.Now(),
	}
}

func (i *Item) updateTimeAccess() {
	i.LastTimeAccess = time.Now()
}

type cacheImpl struct {
	// Key-value store
	kv   map[string]*Item
	kvMu sync.RWMutex

	// Index-key store
	keys map[int]string
	kMu  sync.RWMutex
	// Last index within keys
	lastIdx   int
	lastIdxMu sync.RWMutex
	// Гsed to iterate over keys
	it   int
	itMu sync.RWMutex

	// Free indexes
	freeIds *stack

	// Default TTL
	ttl time.Duration
}

func NewCache(ttl time.Duration) Cache {
	return &cacheImpl{
		kv:      make(map[string]*Item),
		keys:    make(map[int]string),
		freeIds: newStack(),
		ttl:     ttl,
	}
}

func (c *cacheImpl) add(key string, value interface{}) *Item {
	item := NewItem(value)
	c.kvMu.Lock()
	c.kv[key] = item
	c.kvMu.Unlock()

	return item
}

func (c *cacheImpl) Add(key string, value interface{}) *Item {
	c.partialCheckExpiration()
	c.tryAddKey(key)
	item := c.add(key, value)

	return item
}

func (c *cacheImpl) contains(key string) bool {
	c.kvMu.RLock()
	_, ok := c.kv[key]
	c.kvMu.RUnlock()

	return ok
}

func (c *cacheImpl) Contains(key string) bool {
	c.partialCheckExpiration()
	v, ok := c.get(key)
	if ok {
		if ok := c.deleteIfOld(key, v); ok {
			return false
		}

		v.updateTimeAccess()
	}
	return ok
}

func (c *cacheImpl) delete(key string) {
	c.kvMu.Lock()
	delete(c.kv, key)
	c.kvMu.Unlock()
}

func (c *cacheImpl) deleteIfOld(key string, item *Item) bool {
	if time.Since(item.LastTimeAccess) > c.ttl {
		c.delete(key)
		return true
	}

	return false
}

func (c *cacheImpl) Delete(key string) {
	c.partialCheckExpiration()
	c.delete(key)
}

func (c *cacheImpl) get(key string) (*Item, bool) {
	c.kvMu.RLock()
	v, ok := c.kv[key]
	c.kvMu.RUnlock()

	return v, ok
}

func (c *cacheImpl) Get(key string) (*Item, bool) {
	c.partialCheckExpiration()
	v, ok := c.get(key)
	if ok {
		if ok := c.deleteIfOld(key, v); ok {
			return nil, false
		}

		v.updateTimeAccess()
	}

	return v, ok
}

func (c *cacheImpl) GetOrAdd(key string, value interface{}) *Item {
	c.partialCheckExpiration()

	if v, ok := c.get(key); ok {
		if deleted := c.deleteIfOld(key, v); !deleted {
			v.updateTimeAccess()
			return v
		}
	}

	c.tryAddKey(key)
	item := c.add(key, value)
	return item
}

func (c *cacheImpl) Count() int {
	c.kvMu.RLock()
	v := len(c.kv)
	c.kvMu.RUnlock()
	return v
}

func (c *cacheImpl) partialCheckExpiration() {
	it := c.nextIt()

	k, ok := c.getKeyByIdx(it)
	if ok {
		v, ok := c.get(k)

		if ok {
			if time.Since(v.LastTimeAccess) > c.ttl {
				c.delete(k)
				c.deleteKeyByIdx(it)
			}
		} else {
			c.deleteKeyByIdx(it)
		}
	}
}

func (c *cacheImpl) getKeyByIdx(v int) (string, bool) {
	c.kMu.RLock()
	k, ok := c.keys[v]
	c.kMu.RUnlock()

	return k, ok
}

func (c *cacheImpl) getLastIdx() int {
	c.lastIdxMu.RLock()
	idx := c.lastIdx
	c.lastIdxMu.RUnlock()

	return idx
}

func (c *cacheImpl) incLastIdx() {
	c.lastIdxMu.Lock()
	c.lastIdx++
	c.lastIdxMu.Unlock()
}

func (c *cacheImpl) nextIt() int {
	c.lastIdxMu.RLock()
	lastIdx := c.lastIdx
	c.lastIdxMu.RUnlock()

	if lastIdx == 0 {
		return 0
	}

	c.itMu.Lock()
	it := c.it
	c.it = (c.it + 1) % lastIdx
	c.itMu.Unlock()

	return it
}

func (c *cacheImpl) tryAddKey(key string) {
	if c.contains(key) {
		return
	}

	idx, err := c.freeIds.Pop()
	if err == errStackIsEmpty {
		idx = c.getLastIdx()
		c.incLastIdx()
	}

	c.kMu.Lock()
	c.keys[idx] = key
	c.kMu.Unlock()
}

func (c *cacheImpl) deleteKeyByIdx(idx int) {
	c.kMu.Lock()
	delete(c.keys, idx)
	c.kMu.Unlock()
	c.freeIds.Push(idx)
}
