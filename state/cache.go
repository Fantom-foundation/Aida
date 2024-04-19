package state

import (
	"sync"

	cc "github.com/Fantom-foundation/Carmen/go/common"
	"github.com/ethereum/go-ethereum/common"
)

// NewCodeCache creates new instance of CodeCache that stores already retrieved code hashes.
func NewCodeCache(capacity int) *CodeCache {
	if capacity <= 0 {
		return &CodeCache{}
	}

	return &CodeCache{
		cache: cc.NewLruCache[CodeKey, common.Hash](capacity),
		mutex: sync.Mutex{},
	}
}

// CodeKey represents the key for CodeCache.
type CodeKey struct {
	addr common.Address
	code string
}

type CodeCache struct {
	cache cc.Cache[CodeKey, common.Hash]
	mutex sync.Mutex
}

// Get returns code hash for given addr and code.
// If hash does not exist within the cache, it is created and stored.
// This operation is thread-safe.
func (c *CodeCache) Get(addr common.Address, code []byte) common.Hash {
	// cache capacity is nil, hence we do not store anything
	if c.cache == nil {
		return common.Hash(cc.Keccak256(code))
	}
	k := CodeKey{addr: addr, code: string(code)}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	h, exists := c.cache.Get(k)
	if exists {
		return h
	}

	h = common.Hash(cc.Keccak256(code))
	c.set(k, h)
	return h
}

func (c *CodeCache) set(k CodeKey, v common.Hash) {
	c.cache.Set(k, v)
}
