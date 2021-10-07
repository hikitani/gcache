package gcache

import (
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

