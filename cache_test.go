package gcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	c := NewCache(1 * time.Hour)

	c.Add("k1", "hello")
	c.Add("k2", "world")
	assert.Equal(t, 2, c.Count())

	v, ok := c.Get("k1")
	assert.Equal(t, "hello", v.Value().(string))
	assert.Equal(t, true, ok)

	v, ok = c.Get("k2")
	assert.Equal(t, "world", v.Value().(string))
	assert.Equal(t, true, ok)

	v, ok = c.Get("k3")
	assert.Nil(t, v)
	assert.Equal(t, false, ok)
	assert.Equal(t, 2, c.Count())

	assert.Equal(t, true, c.Contains("k1"))
	assert.Equal(t, true, c.Contains("k2"))
	assert.Equal(t, false, c.Contains("k3"))

	c.Delete("k2")
	v, ok = c.Get("k2")
	assert.Equal(t, false, c.Contains("k2"))
	assert.Equal(t, false, ok)
	assert.Equal(t, 1, c.Count())
	assert.Nil(t, v)
}

func TestCacheTtl(t *testing.T) {
	c := NewCache(1 * time.Second)

	c.Add("k1", "hello")
	c.Add("k2", "big")
	c.Add("k3", "world")

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Microsecond)
		for {
			select {
			case <-ticker.C:
				c.Get("k1")
				c.Get("k3")
			case <-done:
				return
			}
		}
	}()

	time.Sleep(2 * time.Second)
	close(done)

	assert.Equal(t, true, c.Contains("k1"))
	assert.Equal(t, false, c.Contains("k2"))
	assert.Equal(t, true, c.Contains("k3"))

	freeIdx, _ := c.freeIds.Top()
	assert.Equal(t, 1, freeIdx)
	assert.Equal(t, 3, c.lastIdx)
	assert.Equal(t, map[int]string{0: "k1", 2: "k3"}, c.keys)

	c.Add("k4", "good")
	assert.Equal(t, true, c.freeIds.IsEmpty())
	assert.Equal(t, 3, c.lastIdx)
	assert.Equal(t, map[int]string{0: "k1", 1: "k4", 2: "k3"}, c.keys)
}

func TestCacheCustomTtl(t *testing.T) {
	c := NewCache(1 * time.Hour)

	c.Add("k1", "hello")
	c.Add("k2", "big").SetTTL(1 * time.Second)
	c.Add("k3", "world")

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Microsecond)
		for {
			select {
			case <-ticker.C:
				c.Get("k1")
				c.Get("k3")
			case <-done:
				return
			}
		}
	}()

	time.Sleep(2 * time.Second)
	close(done)

	assert.Equal(t, true, c.Contains("k1"))
	assert.Equal(t, false, c.Contains("k2"))
	assert.Equal(t, true, c.Contains("k3"))
}

func TestItemConstructor(t *testing.T) {
	c := NewCache(1 * time.Second)
	c.SetItemConstructor(func(key string) interface{} {
		return "default"
	})

	v, ok := c.Get("new key")
	assert.Equal(t, "default", v.Value().(string))
	assert.Equal(t, true, ok)

	c.Add("k1", "hello")
	time.Sleep(2 * time.Second)

	v, ok = c.Get("k1")
	assert.Equal(t, "default", v.Value().(string))
	assert.Equal(t, true, ok)
}

func TestCacheEvictionCB(t *testing.T) {
	c := NewCache(1 * time.Second)

	evictedItems := map[string]*Item{}
	c.OnEvicted(func(key string, v *Item) {
		evictedItems[key] = v
	})

	c.Add("k1", "hello")
	k2 := c.Add("k2", "big")
	k3 := c.Add("k3", "very big")
	c.Add("k4", "world")

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Microsecond)
		for {
			select {
			case <-ticker.C:
				c.Get("k1")
				c.Get("k4")
			case <-done:
				return
			}
		}
	}()

	time.Sleep(2 * time.Second)
	close(done)

	assert.Equal(
		t,
		map[string]*Item{
			"k2": k2,
			"k3": k3,
		},
		evictedItems,
	)
}
