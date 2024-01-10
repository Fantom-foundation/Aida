package register

import (
	"strconv"

	"github.com/Fantom-foundation/Aida/utils"
)

const (
	MetadataCreateTableIfNotExist = `
		CREATE TABLE IF NOT EXISTS metadata (
			key STRING NOT NULL,
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
)

type RunMetadata struct {
	meta map[string]string
	ps   *utils.Printers
}

func MakeRunMetadata(connection string, id *RunIdentity) (*RunMetadata, error) {
	var meta map[string]string = make(map[string]string)

	meta["AppName"] = id.Cfg.AppName
	meta["CommandName"] = id.Cfg.CommandName
	meta["RegisterRun"] = id.Cfg.RegisterRun
	meta["OverwriteRunId"] = id.Cfg.OverwriteRunId

	meta["DbImpl"] = id.Cfg.DbImpl
	meta["DbVariant"] = id.Cfg.DbVariant
	meta["CarmenSchema"] = strconv.Itoa(id.Cfg.CarmenSchema)
	meta["VmImpl"] = id.Cfg.VmImpl
	meta["ArchiveMode"] = strconv.FormatBool(id.Cfg.ArchiveMode)
	meta["ArchiveQueryRate"] = strconv.Itoa(id.Cfg.ArchiveQueryRate)
	meta["ArchiveVariant"] = id.Cfg.ArchiveVariant

	meta["First"] = strconv.Itoa(int(id.Cfg.First))
	meta["Last"] = strconv.Itoa(int(id.Cfg.Last))

	meta["RunId"] = id.GetId()
	meta["Timestamp"] = strconv.Itoa(int(id.Timestamp))

	rm := &RunMetadata{
		meta: meta,
		ps:   utils.NewPrinters(),
	}

	p2db, err := utils.NewPrinterToSqlite3(rm.sqlite3(connection))
	if err != nil {
		return nil, err
	}
	rm.ps.AddPrinter(p2db)

	return rm, nil
}

func (rm *RunMetadata) Print() {
	rm.ps.Print()
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
