package state

//go:generate mockgen -source cache.go -destination cache_mocks.go -package state
import (
	"sync"

	cc "github.com/Fantom-foundation/Carmen/go/common"
	"github.com/ethereum/go-ethereum/common"
)

// CodeCache serves a cache for hashed address code.
type CodeCache interface {
	// Get returns code hash for given addr and code.
	Get(addr common.Address, code []byte) common.Hash
}

// codeKey represents the key for CodeCache.
type codeKey struct {
	addr common.Address
	code string
}

// NewCodeCache creates new instance of CodeCache that stores already retrieved code hashes.
func NewCodeCache(capacity int) CodeCache {
	if capacity <= 0 {
		return &codeCache{}
	}
	return &codeCache{
		cache: cc.NewLruCache[codeKey, common.Hash](capacity),
		mutex: sync.Mutex{},
	}
}

type codeCache struct {
	cache cc.Cache[codeKey, common.Hash]
	mutex sync.Mutex
}

// Get returns code hash for given addr and code.
// If hash does not exist within the cache, it is created and stored.
// This operation is thread-safe.
func (c *codeCache) Get(addr common.Address, code []byte) common.Hash {
	k := codeKey{addr: addr, code: string(code)}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	h, exists := c.cache.Get(k)
	if exists {
		return h
	}

	h = createCodeHash(code)
	c.set(k, h)
	return h
}

func (c *codeCache) set(k codeKey, v common.Hash) {
	c.cache.Set(k, v)
}

func createCodeHash(code []byte) common.Hash {
	return common.Hash(cc.Keccak256(code))
}
