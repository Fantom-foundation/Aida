package db

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
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

	// TODO load stats of current aida-db to download just latest patches
	if true {
		// aida-db already exists appending only new patches
	} else {
		// aida-db does not exist, download all available patches
	}

	// retrieve available patches from aida-db generation server
	patches, err := retrievePatchesToDownload(cfg, log)
	if err != nil {
		return fmt.Errorf("unable to prepare list of aida-db patches for download; %v", err)
	}

	// create a parents of temporary directory
	err = os.MkdirAll(cfg.DbTmp, 0744)
	if err != nil {
		return fmt.Errorf("failed to create %s directory; %s", cfg.DbTmp, err)
	}

	log.Infof("Downloading Aida-db - %d new patches", len(patches))

	// TODO parallelize
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
		cmd := exec.Command("bash", "-c", "tar -xzf "+compressedPatchPath+" -C "+cfg.DbTmp)
		err = runCommand(cmd, nil, log)
		if err != nil {
			return fmt.Errorf("unable extract tar patch %v; %v", compressedPatchPath, err)
		}

		// extracted patch is folder without the .tar.gz extension
		extractedPatchPath := strings.TrimSuffix(compressedPatchPath, ".tar.gz")

		// merge newly extracted patch
		err = Merge(cfg, []string{extractedPatchPath})
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
func retrievePatchesToDownload(cfg *utils.Config, log *logging.Logger) ([]string, error) {
	// download list of available patches
	patches, err := downloadPatchesJson(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to download patches: %v", err)
	}

	// TODO change only few patches may be downloaded not all
	var fileNames = make([]string, len(patches))

	for i, patch := range patches {
		patchMap, ok := patch.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid patch in json; %v", patch)
		}

		fileName, ok := patchMap["fileName"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid attributes in patch; %v", patchMap)
		}
		fileNames[i] = fileName
	}

	// TODO order patches by their sequence - patches.json doesn't have to be ordered

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
	err := os.MkdirAll(parentPath, 0744)
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
