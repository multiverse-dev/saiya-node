package consensus

import (
	"container/list"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// relayCache is a payload cache which is used to store
// last consensus payloads.
type relayCache struct {
	*sync.RWMutex

	maxCap int
	elems  map[common.Hash]*list.Element
	queue  *list.List
}

// hashable is a type of items which can be stored in the relayCache.
type hashable interface {
	Hash() common.Hash
}

func newFIFOCache(capacity int) *relayCache {
	return &relayCache{
		RWMutex: new(sync.RWMutex),

		maxCap: capacity,
		elems:  make(map[common.Hash]*list.Element),
		queue:  list.New(),
	}
}

// Add adds payload into a cache if it doesn't already exist.
func (c *relayCache) Add(p hashable) {
	c.Lock()
	defer c.Unlock()

	h := p.Hash()
	if c.elems[h] != nil {
		return
	}

	if c.queue.Len() >= c.maxCap {
		first := c.queue.Front()
		c.queue.Remove(first)
		delete(c.elems, first.Value.(hashable).Hash())
	}

	e := c.queue.PushBack(p)
	c.elems[h] = e
}

// Has checks if an item is already in cache.
func (c *relayCache) Has(h common.Hash) bool {
	c.RLock()
	defer c.RUnlock()

	return c.elems[h] != nil
}

// Get returns payload with the specified hash from cache.
func (c *relayCache) Get(h common.Hash) hashable {
	c.RLock()
	defer c.RUnlock()

	e, ok := c.elems[h]
	if !ok {
		return hashable(nil)
	}
	return e.Value.(hashable)
}
