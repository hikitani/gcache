package example

import (
	"fmt"
	"testing"
	"time"

	"github.com/hikitani/gcache"
)

func TestExample(t *testing.T) {
	// Create a cache with ttl of 1 second
	c := gcache.NewCache(1 * time.Second)

	c.Add("k1", "hello")
	c.Add("k2", "big")
	c.Add("k3", "world")

	hello, ok := c.Get("k1")
	if ok {
		fmt.Printf("key - k1, value - %s\n", hello.Value.(string))
	}

	done := make(chan struct{})

	// Ttl works as janitor if you have constant access to the cache
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

	if !c.Contains("k2") { // or if _, ok = c.Get("k2"); !ok
		fmt.Println("key k2 doesnt exist")
	}
}
