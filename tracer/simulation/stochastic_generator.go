package simulation

import (
	"crypto/sha256"
	"encoding/binary"
	"github.com/Fantom-foundation/Aida/tracer/dict"
	"github.com/ethereum/go-ethereum/common"
	"math/rand"
)

// GeneratorType specifies the type of generated data
type GeneratorType int

const (
	TContract GeneratorType = iota
	TStorage
	TValue
)

// StochasticGenerator wraps a Label of the distribution and a function to get a next value withing the given distribution
type StochasticGenerator struct {
	DCtx *dict.DictionaryContext
	T    GeneratorType // specifying type of generated data
	C    []float32     // chance for new value for each operation
	Size uint64        // current size of generated items
	E    float64       // exponential rate at which are new values generated
}

var hasher = sha256.New()

// GetNext based on C determines whether new value should be generated or old one used
func (g *StochasticGenerator) GetNext(opId byte) []any {
	nc := rand.Float32()
	if nc <= g.C[opId] || g.Size == 0 {
		//	generating new value
		return g.GetNew()
	} else {
		//	using existing value
		return []any{g.getExisting()}
	}
}

// getExisting retrieves existing value based on distribution
func (g *StochasticGenerator) getExisting() uint64 {
	var expRate float64
	if g.E != 0 {
		expRate = g.E
	} else {
		expRate = float64(10) / float64(g.Size)
	}
	return uint64(rand.ExpFloat64()/expRate) % g.Size
}

// GetNew generates new value and encodes it into dictionary
func (g *StochasticGenerator) GetNew() []any {
	g.Size++
	switch g.T {
	case TContract:
		{
			var address = common.BytesToAddress(RetrieveValueAt(g.Size))
			idx := g.DCtx.EncodeContract(address)
			return []any{idx}
		}
	case TStorage:
		{
			key := common.BytesToHash(RetrieveValueAt(g.Size))
			sIdx, pos := g.DCtx.EncodeStorage(common.BytesToHash(key[:]))

			return []any{sIdx, pos}
		}
	case TValue:
		{
			value := common.BytesToHash(RetrieveValueAt(g.Size))
			idx := g.DCtx.EncodeValue(value)

			return []any{idx}
		}
	default:
		return nil
	}
}

// i64tob convert uint64 to []byte
func i64tob(val uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, val)
	return b
}

// RetrieveValueAt generates 64B data deterministically
func RetrieveValueAt(i uint64) []byte {
	hasher.Reset()
	hasher.Write(i64tob(i))
	return hasher.Sum(nil)
}
