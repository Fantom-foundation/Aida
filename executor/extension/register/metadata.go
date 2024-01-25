package register

//go:generate mockgen -source metadata.go -destination metadata_mocks.go -package register

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
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

// MakeRunMetadata creates RunMetadata and write it into the DB.
// 1. collect run config, timestamp and app name.
// 2. fetch environment information about where the run is executed.
// On Print(), print all metadata into the corresponding table.
func MakeRunMetadata(connection string, id *RunIdentity) (*RunMetadata, error) {
	rm := &RunMetadata{
		meta: make(map[string]string),
		ps:   utils.NewPrinters(),
	}

	rm.meta["AppName"] = id.Cfg.AppName
	rm.meta["CommandName"] = id.Cfg.CommandName
	rm.meta["RegisterRun"] = id.Cfg.RegisterRun
	rm.meta["OverwriteRunId"] = id.Cfg.OverwriteRunId

	rm.meta["DbImpl"] = id.Cfg.DbImpl
	rm.meta["DbVariant"] = id.Cfg.DbVariant
	rm.meta["CarmenSchema"] = strconv.Itoa(id.Cfg.CarmenSchema)
	rm.meta["VmImpl"] = id.Cfg.VmImpl
	rm.meta["ArchiveMode"] = strconv.FormatBool(id.Cfg.ArchiveMode)
	rm.meta["ArchiveQueryRate"] = strconv.Itoa(id.Cfg.ArchiveQueryRate)
	rm.meta["ArchiveVariant"] = id.Cfg.ArchiveVariant

	rm.meta["First"] = strconv.Itoa(int(id.Cfg.First))
	rm.meta["Last"] = strconv.Itoa(int(id.Cfg.Last))

	rm.meta["RunId"] = id.GetId()
	rm.meta["Timestamp"] = strconv.Itoa(int(id.Timestamp))

	p2db, err := utils.NewPrinterToSqlite3(rm.sqlite3(connection))
	if err != nil {
		return nil, err
	}
	rm.ps.AddPrinter(p2db)

	// commands that failed are to be logged, but they are not fatal.
	warnings := rm.fetchEnvInfo()
	return rm, warnings
}

func (rm *RunMetadata) Print() {
	rm.ps.Print()
}

func (rm *RunMetadata) Close() {
	rm.ps.Close()
}

type EnvInfoFetcher interface {
	FetchEnvInfo() error
}

// fetchEnvInfo fetches environment info by executing a number of linux commands.
// Any errors are collected and returned.
func (rm *RunMetadata) FetchEnvInfo() error {
	var errs error
	for tag, f := range map[string]func() (string, error){
		"Processor":     rm.GetProcessor,
		"Memory":        rm.GetMemory,
		"Disks":         rm.GetDisks,
		"Os":            rm.GetOs,
		"AidaGitHash":   rm.GetAidaGitHash,
		"CarmenGitHash": rm.GetCarmenGitHash,
		"ToscaGitHash":  rm.GetToscaGitHash,
		"GoVersion":     rm.GetGoVersion,
		"Hostname":      rm.GetHostname,
		"IpAddress":     rm.GetIpAddress,
	} {
		out, err := f()
		if err != nil {
			errs = errors.Join(errs, errors.New(fmt.Sprintf("Couldn't get %s: %s.", tag, err)))
		}
		rm.meta[tag] = out
	}

	return errs
}

func (rm *RunMetadata) bash(cmd string) (string, error) {
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (rm *RunMetadata) GetProcessor() (string, error) {
	return rm.bash(bashCmdProcessor)
}

func (rm *RunMetadata) GetMemory() (string, error) {
	return rm.bash(bashCmdMemory)
}

func (rm *RunMetadata) GetDisks() (string, error) {
	return rm.bash(bashCmdDisks)
}

func (rm *RunMetadata) GetOs() (string, error) {
	return rm.bash(bashCmdOs)
}

func (rm *RunMetadata) GetAidaGitHash() (string, error) {
	return rm.bash(bashCmdAidaGitHash)
}

func (rm *RunMetadata) GetCarmenGitHash() (string, error) {
	return rm.bash(bashCmdCarmenGitHash)
}

func (rm *RunMetadata) GetToscaGitHash() (string, error) {
	return rm.bash(bashCmdToscaGitHash)
}

func (rm *RunMetadata) GetGoVersion() (string, error) {
	return rm.bash(bashCmdGoVersion)
}

func (rm *RunMetadata) GetHostname() (string, error) {
	return rm.bash(bashCmdHostname)
}

func (rm *RunMetadata) GetIpAddress() (string, error) {
	return rm.bash(bashCmdIpAddress)
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
