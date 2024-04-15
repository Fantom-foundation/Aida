// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
		"ChainId":          strconv.Itoa(int(id.Cfg.ChainID)),

		"DbSrc":         id.Cfg.StateDbSrc,
		"RpcRecordings": id.Cfg.RpcRecordingPath,

		"First": strconv.Itoa(int(id.Cfg.First)),
		"Last":  strconv.Itoa(int(id.Cfg.Last)),

		"RunId":     id.GetId(),
		"Timestamp": strconv.Itoa(int(id.Timestamp)),
	}

	return info, nil
}
