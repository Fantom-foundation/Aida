package register

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"

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

func (id *RunIdentity) fetchConfigInfo() (map[string]string, error) {
	info := map[string]string{
		"AppName":        id.Cfg.AppName,
		"CommandName":    id.Cfg.CommandName,
		"RegisterRun":    id.Cfg.RegisterRun,
		"OverwriteRunId": id.Cfg.OverwriteRunId,

		"DbImpl":           id.Cfg.DbImpl,
		"DbVariant":        id.Cfg.DbVariant,
		"CarmenSchema":     strconv.Itoa(id.Cfg.CarmenSchema),
		"VmImpl":           id.Cfg.VmImpl,
		"ArchiveMode":      strconv.FormatBool(id.Cfg.ArchiveMode),
		"ArchiveQueryRate": strconv.Itoa(id.Cfg.ArchiveQueryRate),
		"ArchiveVariant":   id.Cfg.ArchiveVariant,

		"First": strconv.Itoa(int(id.Cfg.First)),
		"Last":  strconv.Itoa(int(id.Cfg.Last)),

		"RunId":     id.GetId(),
		"Timestamp": strconv.Itoa(int(id.Timestamp)),
	}

	return info, nil
}
