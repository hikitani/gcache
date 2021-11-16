package gcache

import (
	"sync"
	"time"
)

type keyValueTuple struct {
	Key   string
	Value *Item
}

type Item struct {
	value      interface{}
	lastAccess time.Time
	created    time.Time
	ttl        time.Duration

	sync.RWMutex
}

func NewItem(v interface{}, ttl time.Duration) *Item {
	now := time.Now()
	return &Item{
		value:      v,
		lastAccess: now,
		created:    now,
		ttl:        ttl,
	}
}

func (i *Item) Value() interface{} {
	return i.value
}

func (i *Item) TTL() time.Duration {
	i.RLock()
	v := i.ttl
	i.RUnlock()
	return v
}

func (i *Item) SetTTL(d time.Duration) {
	i.Lock()
	i.ttl = d
	i.Unlock()
}

func (i *Item) LastAccess() time.Time {
	return i.lastAccess
}

func (i *Item) Expired() bool {
	return time.Since(i.LastAccess()) > i.TTL()
}

func (i *Item) updateAccess() {
	i.lastAccess = time.Now()
}

type Cache struct {
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

	createItem    func(key string) interface{}
	evictionItems chan *keyValueTuple
	evMu          sync.RWMutex

	sync.RWMutex
}

func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		kv:      make(map[string]*Item),
		keys:    make(map[int]string),
		freeIds: newStack(),
		ttl:     ttl,
	}
}

// Sets item constructor, which called when trying to access a non-existing key (Optional).
func (c *Cache) SetItemConstructor(f func(key string) interface{}) {
	c.Lock()
	c.createItem = f
	c.Unlock()
}

// Sets callback, which called when item is evicted from cache (Optional).
func (c *Cache) OnEvicted(f func(key string, v *Item)) {
	if c.evictionItems != nil {
		chanNotClosed := true
		select {
		case _, chanNotClosed = <-c.evictionItems:
		default:
		}

		if chanNotClosed {
			close(c.evictionItems)
		}
	}

	c.evMu.Lock()
	c.evictionItems = make(chan *keyValueTuple, 1000)
	c.evMu.Unlock()
	go func() {
		for kv := range c.evictionItems {
			f(kv.Key, kv.Value)
		}
	}()
}

// Adds value by key with default TTL and returns item object.
// Use SetTTL of item for change default TTL.
func (c *Cache) Add(key string, value interface{}) *Item {
	c.partialCheckExpiration()
	c.tryAddKey(key)
	item := c.add(key, value)

	return item
}

// Returns true if key exists.
func (c *Cache) Contains(key string) bool {
	c.partialCheckExpiration()
	v, ok := c.get(key)
	if ok {
		if ok := c.deleteIfOld(key, v); ok {
			return false
		}

		v.updateAccess()
	}
	return ok
}

// Deletes item from cache by key.
func (c *Cache) Delete(key string) {
	c.partialCheckExpiration()
	c.delete(key)
}

// Returns item and boolean value for check of key existings.
// If key doesn't exist and item constructor is defined then item constructor is called.
func (c *Cache) Get(key string) (*Item, bool) {
	c.partialCheckExpiration()
	v, ok := c.get(key)
	var deleted bool
	if ok {
		deleted = c.deleteIfOld(key, v)

		if !deleted {
			v.updateAccess()
		}
	}

	if !ok || deleted {
		c.RLock()
		createItem := c.createItem
		c.RUnlock()
		if createItem == nil {
			return nil, false
		}

		data := createItem(key)
		if data == nil {
			return nil, false
		}
		v = c.Add(key, data)
		ok = true
	}

	return v, ok
}

// Returns count of items in the cache
func (c *Cache) Count() int {
	c.kvMu.RLock()
	v := len(c.kv)
	c.kvMu.RUnlock()
	return v
}

func (c *Cache) add(key string, value interface{}) *Item {
	item := NewItem(value, c.ttl)
	c.kvMu.Lock()
	c.kv[key] = item
	c.kvMu.Unlock()

	return item
}

func (c *Cache) contains(key string) bool {
	c.kvMu.RLock()
	_, ok := c.kv[key]
	c.kvMu.RUnlock()

	return ok
}

func (c *Cache) delete(key string) {
	v, _ := c.get(key)

	c.kvMu.Lock()
	_, ok := c.kv[key]
	delete(c.kv, key)
	c.kvMu.Unlock()

	if ok && v != nil && c.evictionItems != nil {
		c.evMu.RLock()
		c.evictionItems <- &keyValueTuple{
			Key:   key,
			Value: v,
		}
		c.evMu.RUnlock()
	}
}

func (c *Cache) deleteIfOld(key string, v *Item) bool {
	if v.Expired() {
		c.delete(key)
		return true
	}

	return false
}

func (c *Cache) get(key string) (*Item, bool) {
	c.kvMu.RLock()
	v, ok := c.kv[key]
	c.kvMu.RUnlock()

	return v, ok
}

func (c *Cache) partialCheckExpiration() {
	it := c.nextIt()

	if k, ok := c.getKeyByIdx(it); !ok {
		return
	} else if v, ok := c.get(k); !ok {
		c.deleteKeyByIdx(it)
	} else if v.Expired() {
		c.delete(k)
		c.deleteKeyByIdx(it)
	}
}

func (c *Cache) getKeyByIdx(v int) (string, bool) {
	c.kMu.RLock()
	k, ok := c.keys[v]
	c.kMu.RUnlock()

	return k, ok
}

func (c *Cache) getLastIdx() int {
	c.lastIdxMu.RLock()
	idx := c.lastIdx
	c.lastIdxMu.RUnlock()

	return idx
}

func (c *Cache) incLastIdx() {
	c.lastIdxMu.Lock()
	c.lastIdx++
	c.lastIdxMu.Unlock()
}

func (c *Cache) nextIt() int {
	lastIdx := c.getLastIdx()

	if lastIdx == 0 {
		return 0
	}

	c.itMu.Lock()
	it := c.it
	c.it = (c.it + 1) % lastIdx
	c.itMu.Unlock()

	return it
}

func (c *Cache) tryAddKey(key string) {
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

func (c *Cache) deleteKeyByIdx(idx int) {
	c.kMu.Lock()
	delete(c.keys, idx)
	c.kMu.Unlock()
	c.freeIds.Push(idx)
}
