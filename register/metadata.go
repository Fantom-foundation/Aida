// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package register

//go:generate mockgen -source metadata.go -destination metadata_mocks.go -package register

import (
	"errors"
	"fmt"
	"maps"
	"os/exec"
	"strings"

	"github.com/Fantom-foundation/Aida/utils"
)

const (
	metadataCreateTableIfNotExist = `
		CREATE TABLE IF NOT EXISTS metadata (
			key STRING NOT NULL PRIMARY KEY,
			value STRING NUT NULL
		)
	`

	metadataInsertOrReplace = `
		INSERT or REPLACE INTO metadata (
			key, value
		) VALUES (
			?, ?
		)
	`
	bashCmdProcessor     = "cat /proc/cpuinfo | grep \"^model name\" | head -n 1 | awk -F': ' '{print $2}'"
	bashCmdMemory        = "free | grep \"^Mem:\" | awk '{printf(\"%dGb RAM\", $2/1024/1024)}'"
	bashCmdDisks         = "hwinfo --disk | grep Model | awk -F': \\\"' '{if (NR > 1) printf(\", \"); printf(\"%s\", substr($2,1,length($2)-1));} END {(\"\\n\")}'"
	bashCmdOs            = "source /etc/*-release; echo $DISTRIB_DESCRIPTION"
	bashCmdAidaGitHash   = "git rev-parse HEAD"
	bashCmdCarmenGitHash = "git submodule--helper list | grep \"carmen\" | awk -F' ' '{print $2'}"
	bashCmdToscaGitHash  = "git submodule--helper list | grep \"tosca\" | awk -F' ' '{print $2'}"
	bashCmdGoVersion     = "go version"
	bashCmdHostname      = "hostname"
	bashCmdIpAddress     = "curl -s api.ipify.org"
)

type RunMetadata struct {
	Meta map[string]string
	ps   *utils.Printers
}

type FetchInfo func() (map[string]string, error)

func MakeRunMetadata(connection string, id *RunIdentity, fetchEnv FetchInfo) (*RunMetadata, error) {
	return makeRunMetadata(connection, id.fetchConfigInfo, fetchEnv)
}

// makeRunMetadata creates RunMetadata to keep track of metadata about the run.
// 1. collect run config, timestamp and app name.
// 2. fetch environment information about where the run is executed.
// 3. On Print(), print all metadata into the corresponding table.
func makeRunMetadata(connection string, fetchCfg FetchInfo, fetchEnv FetchInfo) (*RunMetadata, error) {
	rm := &RunMetadata{
		Meta: make(map[string]string),
		ps:   utils.NewPrinters(),
	}

	var warnings error

	// 1. collect run config, timestamp and app name.
	cfgInfo, w := fetchCfg()
	if w != nil {
		// commands that failed are to be logged, but they are not fatal.
		warnings = errors.Join(warnings, w)
	}
	maps.Copy(rm.Meta, cfgInfo)

	// 2. fetch environment information about where the run is executed.
	envInfo, w := fetchEnv()
	if w != nil {
		// commands that failed are to be logged, but they are not fatal.
		warnings = errors.Join(warnings, w)
	}
	maps.Copy(rm.Meta, envInfo)

	// 3. On Print(), print all metadata into the corresponding table.
	p2db, err := utils.NewPrinterToSqlite3(rm.sqlite3(connection))
	if err != nil {
		return nil, err
	}
	rm.ps.AddPrinter(p2db)

	return rm, warnings
}

func (rm *RunMetadata) Print() {
	rm.ps.Print()
}

func (rm *RunMetadata) Close() {
	rm.ps.Close()
}

// fetchEnvInfo fetches environment info by executing a number of linux commands.
// Any errors are collected and returned.
func FetchUnixInfo() (map[string]string, error) {
	cmds := map[string]func() (string, error){
		"Processor":     func() (string, error) { return bash(bashCmdProcessor) },
		"Memory":        func() (string, error) { return bash(bashCmdMemory) },
		"Disks":         func() (string, error) { return bash(bashCmdDisks) },
		"Os":            func() (string, error) { return bash(bashCmdOs) },
		"AidaGitHash":   func() (string, error) { return bash(bashCmdAidaGitHash) },
		"CarmenGitHash": func() (string, error) { return bash(bashCmdCarmenGitHash) },
		"ToscaGitHash":  func() (string, error) { return bash(bashCmdToscaGitHash) },
		"GoVersion":     func() (string, error) { return bash(bashCmdGoVersion) },
		"Hostname":      func() (string, error) { return bash(bashCmdHostname) },
		"IpAddress":     func() (string, error) { return bash(bashCmdIpAddress) },
	}

	envs := make(map[string]string, len(cmds))
	var errs error
	for tag, f := range cmds {
		out, err := f()
		if err != nil {
			errs = errors.Join(errs, errors.New(fmt.Sprintf("bash cmd failed to get %s; %v.", tag, err)))
		}
		envs[tag] = out
	}
	return envs, errs
}

func bash(cmd string) (string, error) {
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (rm *RunMetadata) sqlite3(conn string) (string, string, string, func() [][]any) {
	return conn, metadataCreateTableIfNotExist,
		metadataInsertOrReplace,
		func() [][]any {
			values := [][]any{}
			for k, v := range rm.Meta {
				values = append(values, []any{k, v})
			}
			return values
		}
}
