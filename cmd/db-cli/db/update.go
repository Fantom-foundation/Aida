package db

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/klauspost/compress/gzip"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

// UpdateCommand downloads aida-db and new patches
var UpdateCommand = cli.Command{
	Action: update,
	Name:   "update",
	Usage:  "download aida-db patches",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&logger.LogLevelFlag,
		&utils.CompactDbFlag,
		&utils.DbTmpFlag,
	},
	Description: ` 
Updates aida-db by downloading patches from aida-db generation server.
`,
}

// update updates aida-db by downloading patches from aida-db generation server.
func update(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}
	return Update(cfg)
}

// Update implements updating command to be called from various commands and automatically downloads aida-db patches.
func Update(cfg *utils.Config) error {
	log := logger.NewLogger(cfg.LogLevel, "DB Update")

	var startDownloadFromBlock uint64

	mdi := new(MetadataInfo)
	mdi.dbType = updateType

	// load stats of current aida-db to download just latest patches
	_, err := os.Stat(cfg.AidaDb)
	if os.IsNotExist(err) {
		// aida-db does not exist, download all available patches
		startDownloadFromBlock = 0
	} else {
		// aida-db already exists appending only new patches
		// open targetDB
		targetDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", true)
		if err != nil {
			return fmt.Errorf("targetDb; %v", err)
		}

		// TODO retrieve metadata
		// and set correct startDownloadFromEpoch from metadata
		startDownloadFromBlock = 0

		// close target database
		MustCloseDB(targetDb)
	}

	// retrieve available patches from aida-db generation server
	patches, err := retrievePatchesToDownload(cfg, startDownloadFromBlock, log)
	if err != nil {
		return fmt.Errorf("unable to prepare list of aida-db patches for download; %v", err)
	}

	// create a parents of temporary directory
	err = os.MkdirAll(cfg.DbTmp, 0755)
	if err != nil {
		return fmt.Errorf("failed to create %s directory; %s", cfg.DbTmp, err)
	}

	log.Infof("Downloading Aida-db - %d new patches", len(patches))

	for _, fileName := range patches {
		log.Debugf("Downloading %s...", fileName)
		patchUrl := utils.AidaDbRepositoryUrl + "/" + fileName
		compressedPatchPath := filepath.Join(cfg.DbTmp, fileName)
		err := downloadFile(compressedPatchPath, cfg.DbTmp, patchUrl)
		if err != nil {
			return fmt.Errorf("unable to download %s; %v", patchUrl, err)
		}
		log.Debugf("Downloaded %s", fileName)

		log.Debugf("Decompressing %v", fileName)

		err = extractTarGz(compressedPatchPath, cfg.DbTmp)
		if err != nil {
			return fmt.Errorf("unable to extract %s; %v", compressedPatchPath, err)
		}

		// extracted patch is folder without the .tar.gz extension
		extractedPatchPath := strings.TrimSuffix(compressedPatchPath, ".tar.gz")

		// merge newly extracted patch
		err = Merge(cfg, []string{extractedPatchPath}, mdi)
		if err != nil {
			return fmt.Errorf("unable to merge %v; %v", extractedPatchPath, err)
		}

		// remove compressed patch
		err = os.RemoveAll(compressedPatchPath)
		if err != nil {
			return err
		}

		// remove patch
		err = os.RemoveAll(extractedPatchPath)
		if err != nil {
			return err
		}
	}

	log.Notice("Aida-db update finished successfully")

	return nil
}

// retrievePatchesToDownload retrieves all available patches from aida-db generation server.
func retrievePatchesToDownload(cfg *utils.Config, startDownloadFromBlock uint64, log *logging.Logger) ([]string, error) {
	// download list of available patches
	patches, err := downloadPatchesJson(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to download patches: %v", err)
	}

	// list of patches to be downloaded
	var fileNames = make([]string, len(patches))

	for i, patch := range patches {
		patchMap, ok := patch.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid patch in json; %v", patch)
		}

		// retrieve fromBlock start of patch
		patchFromBlockStr, ok := patchMap["fromBlock"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid fromEpoch attributes in patch; %v", patchMap)
		}
		patchFromBlock, err := strconv.ParseUint(patchFromBlockStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse uint64 fromEpoch attribute in patch %v; %v", patchMap, err)
		}
		if patchFromBlock <= startDownloadFromBlock {
			continue
		}

		fileName, ok := patchMap["fileName"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid fileName attributes in patch; %v", patchMap)
		}
		fileNames[i] = fileName
	}

	sort.Strings(fileNames)

	return fileNames, nil
}

// downloadPatchesJson downloads list of available patches from aida-db generation server.
func downloadPatchesJson(cfg *utils.Config) ([]interface{}, error) {
	// Make the HTTP GET request
	patchesUrl := utils.AidaDbRepositoryUrl + "/patches.json"
	response, err := http.Get(patchesUrl)
	if err != nil {
		return nil, fmt.Errorf("error making GET request for %s: %v", patchesUrl, err)
	}
	defer response.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Parse the JSON data
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response body %v: %v", body, err)
	}

	// Access the JSON data
	return data.([]interface{}), nil
}

// downloadFile downloads file - used for downloading individual patches.
func downloadFile(filePath string, parentPath string, url string) error {
	// Create parent directories if they don't exist
	err := os.MkdirAll(parentPath, 0755)
	if err != nil {
		return fmt.Errorf("error creating parent directories: %v", err)
	}

	// Create the file
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s, bad status: %s", url, resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// extractTarGz extracts tar file contents into location of output folder
func extractTarGz(tarGzFile, outputFolder string) error {
	// Open the tar.gz file
	file, err := os.Open(tarGzFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create the gzip readerÏ
	gr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gr.Close()

	// Create the tar reader
	tr := tar.NewReader(gr)

	// Extract the files from the tar reader
	for {
		header, err := tr.Next()
		if err == io.EOF {
			// Reached the end of the tar archive
			break
		}
		if err != nil {
			return err
		}

		// Determine the output file path
		targetPath := filepath.Join(outputFolder, header.Name)

		// Check if it's a directory
		if header.FileInfo().IsDir() {
			// Create the directory
			err = os.MkdirAll(targetPath, 0755)
			if err != nil {
				return err
			}
		} else {
			// Create the parent directory of the file
			err = os.MkdirAll(filepath.Dir(targetPath), 0755)
			if err != nil {
				return err
			}

			// Create the output file
			file, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			defer file.Close()

			// Copy the file content from the tar reader to the output file
			_, err = io.Copy(file, tr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
