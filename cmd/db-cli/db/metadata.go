package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const PathToAidaDbInfo = "aida-db_info.json"

type metadata struct {
	first, last uint64
	createTime  string
}

func createMetaDataFile(directory string, blockStart, blockEnd uint64) error {
	filename := filepath.Join(directory, PathToAidaDbInfo)

	// remove file if exists
	os.RemoveAll(filename)

	dbInfo := &metadata{
		first:      blockStart,
		last:       blockEnd,
		createTime: time.Now().UTC().Format(time.UnixDate),
	}

	jsonByte, err := json.MarshalIndent(dbInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal AidaDbInfo, cmd was successful but metadata file was not creater; %v", err)
	}

	if err = os.WriteFile(filename, jsonByte, 0666); err != nil {
		return fmt.Errorf("cannot create file %v, cmd was successful but metadata file was not created; %v", filename, err)
	}
	return nil
}
