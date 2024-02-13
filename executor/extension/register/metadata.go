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
	MetadataCreateTableIfNotExist = `
		CREATE TABLE IF NOT EXISTS metadata (
			key STRING NOT NULL PRIMARY KEY,
			value STRING NUT NULL
		)
	`

	MetadataInsertOrReplace = `
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
	bashCmdCarmenGitHash = "git submodule status | grep \"carmen\" | awk -F' ' '{print $1'}"
	bashCmdToscaGitHash  = "git submodule status | grep \"tosca\" | awk -F' ' '{print $1'}"
	bashCmdGoVersion     = "go version"
	bashCmdHostname      = "hostname"
	bashCmdIpAddress     = "curl -s api.ipify.org"
)

type RunMetadata struct {
	meta map[string]string
	ps   *utils.Printers
}

type FetchInfo func() (map[string]string, error)

func MakeRunMetadata(connection string, id *RunIdentity) (*RunMetadata, error) {
	return makeRunMetadata(connection, id.fetchConfigInfo, fetchUnixInfo)
}

// makeRunMetadata creates RunMetadata to keep track of metadata about the run.
// 1. collect run config, timestamp and app name.
// 2. fetch environment information about where the run is executed.
// 3. On Print(), print all metadata into the corresponding table.
func makeRunMetadata(connection string, fetchCfg FetchInfo, fetchEnv FetchInfo) (*RunMetadata, error) {
	rm := &RunMetadata{
		meta: make(map[string]string),
		ps:   utils.NewPrinters(),
	}

	var warnings error

	// 1. collect run config, timestamp and app name.
	cfgInfo, w := fetchCfg()
	if w != nil {
		// commands that failed are to be logged, but they are not fatal.
		warnings = errors.Join(warnings, w)
	}
	maps.Copy(rm.meta, cfgInfo)

	// 2. fetch environment information about where the run is executed.
	envInfo, w := fetchEnv()
	if w != nil {
		// commands that failed are to be logged, but they are not fatal.
		warnings = errors.Join(warnings, w)
	}
	maps.Copy(rm.meta, envInfo)

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
func fetchUnixInfo() (map[string]string, error) {
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
	return conn, MetadataCreateTableIfNotExist, MetadataInsertOrReplace,
		func() [][]any {
			values := [][]any{}
			for k, v := range rm.meta {
				values = append(values, []any{k, v})
			}
			return values
		}
}
