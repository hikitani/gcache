package gcache

import (
	"sync"
	"time"
)

type Cache interface {
	Get(key string) (Item, error)
	Add(key string, value interface{})
	Delete(key string)
	GetOrAdd(key string, value interface{})
	Contains(key string) bool
}

type Item struct {
	Value      interface{}
	LastAccess time.Time
}

func NewItem(v interface{}) *Item {
	return &Item{
		Value:      v,
		LastAccess: time.Now(),
	}
}

type cacheImpl struct {
	// Хранилище значений
	kv   map[string]*Item
	kvMu sync.RWMutex

	// Хранилище ключей, который используется для перебора по индексу
	keys map[int]string
	kMu  sync.RWMutex
	// Хранит последний индекс в хранилище ключей
	idx   int
	idxMu sync.RWMutex
	// Перебираемый индекс в хранилище ключей
	it   int
	itMu sync.RWMutex

	// Свободные индексы
	freeIds *stack

	ttl time.Duration
}

func (c *cacheImpl) add(key string, value interface{}) {
	item := NewItem(value)
	c.kvMu.Lock()
	c.kv[key] = item
	c.kvMu.Unlock()
}

func (c *cacheImpl) Add(key string, value interface{}) {
	c.add(key, value)

	// TODO: check evict

	c.idxMu.RLock()
	idx := c.idx
	c.idxMu.RUnlock()

	if !c.freeIds.IsEmpty() {
		idx, _ = c.freeIds.Pop()
	} else {
		// Инкрементим только если свободных индексов нет
		c.idxMu.Lock()
		c.idx++
		c.idxMu.Unlock()
	}
	c.kMu.Lock()
	c.keys[idx] = key
	c.kMu.Unlock()
}

func (c *cacheImpl) contains(key string) bool {
	c.kvMu.RLock()
	_, ok := c.kv[key]
	c.kvMu.RUnlock()

	return ok
}

func (c *cacheImpl) Contains(key string) bool {
	ok := c.contains(key)

	// TODO: check evict

	return ok
}

func (c *cacheImpl) delete(key string) {
	c.kvMu.Lock()
	delete(c.kv, key)
	c.kvMu.Unlock()
}

func (c *cacheImpl) Delete(key string) {
	c.delete(key)

	c.itMu.Lock()
	it := c.it
	c.it = (c.it + 1) % (c.idx + 1)
	c.itMu.Unlock()

	c.kMu.RLock()
	k, ok := c.keys[it]
	c.kMu.RUnlock()
	if ok {
		c.kvMu.RLock()
		v, ok := c.kv[k]
		c.kvMu.RUnlock()

		if ok {
			if time.Since(v.LastAccess) > c.ttl {
				c.delete(k)
				c.freeIds.Push(it)
			}
		} else {
			c.kMu.Lock()
			delete(c.keys, it)
			c.kMu.Unlock()
		}
	}
}
