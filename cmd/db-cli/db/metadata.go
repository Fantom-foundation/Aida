package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/op/go-logging"
)

const PathToAidaDbInfo = "aida-db_info.json"

type metadata struct {
	epochRange    epochRange
	updatesetInfo updateset
	createTime    string
}

type epochRange struct {
	first, last uint64
}

type updateset struct {
	interval, size uint64
}

func createMetaDataFile(log *logging.Logger, directory string, epochStart, epochEnd, updatesetInterval, updatesetSize uint64) error {
	dbInfo := &metadata{
		epochRange: epochRange{
			first: epochStart,
			last:  epochEnd,
		},
		updatesetInfo: updateset{
			interval: updatesetInterval,
			size:     updatesetSize,
		},
		createTime: time.Now().UTC().Format(time.UnixDate),
	}

	filename := filepath.Join(directory, PathToAidaDbInfo)
	jsonByte, err := json.MarshalIndent(dbInfo, "", "  ")
	if err != nil {
		log.Errorf("cannot marshal AidaDbInfo, cmd was successful but metadata file was not creater; %v", err)
	}

	if err = os.WriteFile(filename, jsonByte, 0666); err != nil {
		log.Errorf("cannot create file %v, cmd was successful but metadata file was not created; %v", filename, err)
	}
	return nil
}
