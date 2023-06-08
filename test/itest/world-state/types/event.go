package types

import "github.com/ethereum/go-ethereum/common"

// EventInfo holds basic information about a consensus Event.
type EventInfo struct {
	ID           common.Hash
	GasPowerLeft GasPowerLeft
	Time         uint64
}
