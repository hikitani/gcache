# gcache

This is a cache with ttl, which does not use an object expiration check loop, which is periodically called and blocks access to the storage. Instead, checking the expiration of the object occurs partially when calling one of the methods in the cache. This allows you not to block access to the storage for a long time. **But it should be understood that ttl will work correctly with a large number of cache accesses**.

Is thread-safe.

## Usage

```go
package main

import (
	"fmt"
	"time"

	"github.com/hikitani/gcache"
)

func main() {
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
```