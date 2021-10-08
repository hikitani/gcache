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

	v, ok := c.Get("k1")
	assert.Equal(t, "hello", v.Value.(string))
	assert.Equal(t, true, ok)

	v, ok = c.Get("k2")
	assert.Equal(t, "world", v.Value.(string))
	assert.Equal(t, true, ok)

	v, ok = c.Get("k3")
	assert.Nil(t, v)
	assert.Equal(t, false, ok)

	assert.Equal(t, true, c.Contains("k1"))
	assert.Equal(t, true, c.Contains("k2"))
	assert.Equal(t, false, c.Contains("k3"))

	v = c.GetOrAdd("k3", "default")
	assert.Equal(t, "default", v.Value)

	c.Delete("k3")
	v, ok = c.Get("k3")
	assert.Equal(t, false, c.Contains("k3"))
	assert.Equal(t, false, ok)
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

	freeIdx, _ := c.(*cacheImpl).freeIds.Top()
	assert.Equal(t, 1, freeIdx)
	assert.Equal(t, 3, c.(*cacheImpl).lastIdx)
	assert.Equal(t, map[int]string{0: "k1", 2: "k3"}, c.(*cacheImpl).keys)

	c.Add("k4", "good")
	assert.Equal(t, true, c.(*cacheImpl).freeIds.IsEmpty())
	assert.Equal(t, 3, c.(*cacheImpl).lastIdx)
	assert.Equal(t, map[int]string{0: "k1", 1: "k4", 2: "k3"}, c.(*cacheImpl).keys)
}
