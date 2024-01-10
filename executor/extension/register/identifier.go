package register

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/Fantom-foundation/Aida/utils"
)

type RunIdentity struct {
	Timestamp int64
	Cfg       *utils.Config
}

func MakeRunIdentity(t int64, cfg *utils.Config) *RunIdentity {
	return &RunIdentity{
		Timestamp: t,
		Cfg:       cfg,
	}
}

func (id *RunIdentity) GetId() string {
	if id.Cfg.OverwriteRunId != "" {
		return id.Cfg.OverwriteRunId
	}
	return id.hash()
}

func (id *RunIdentity) hash() string {
	var b bytes.Buffer

	gob.NewEncoder(&b).Encode([]string{
		fmt.Sprintf("%d", id.Timestamp),
		id.Cfg.DbImpl,
		id.Cfg.DbVariant,
		fmt.Sprintf("%d", id.Cfg.CarmenSchema),
		id.Cfg.VmImpl,
		fmt.Sprintf("%d", id.Cfg.First),
		fmt.Sprintf("%d", id.Cfg.Last),
	})

	return b.String()
}
